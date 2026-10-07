// FunChat iOS - 聊天详情页 (CHAT-009)
// 消息气泡 + 输入框 + WebSocket 实时收发
// 基于 DES-003 设计稿 P2

import SwiftUI
import UIKit

// MARK: - WebSocket 客户端

class WebSocketManager: ObservableObject {
    @Published var messages: [ChatBubbleMessage] = []
    @Published var isConnected = false
    @Published var typingUser: String?

    private var webSocketTask: URLSessionWebSocketTask?
    private let session = URLSession.shared

    struct ChatBubbleMessage: Identifiable {
        let id: Int64
        let fromUserID: String
        let toUserID: String
        let msgType: Int8       // 1=文本 2=图片
        let content: String
        let isBurn: Bool
        let burnDuration: Int
        let timestamp: Int64
        var isMe: Bool { fromUserID == Self.currentUserID }
        var sendStatus: SendStatus = .sent  // BUG-003 修复: 发送状态

        enum SendStatus {
            case sending    // 发送中（半透明）
            case sent       // 发送成功
            case failed     // 发送失败（红色感叹号，可重试）
        }

        static var currentUserID: String = ""
    }

    func connect(token: String, userID: String) {
        ChatBubbleMessage.currentUserID = userID
        let url = URL(string: "ws://localhost:8080/ws?token=\(token)")!
        webSocketTask = session.webSocketTask(with: url)
        webSocketTask?.resume()
        isConnected = true
        receive()
        startPing()
    }

    func disconnect() {
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        isConnected = false
    }

    func send(message: WSOutgoingMessage) {
        guard let data = try? JSONEncoder().encode(message),
              let string = String(data: data, encoding: .utf8) else { return }
        webSocketTask?.send(.string(string)) { _ in }
    }

    private func receive() {
        webSocketTask?.receive { [weak self] result in
            switch result {
            case .success(let message):
                switch message {
                // URLSessionWebSocketTask.Message 的 case 是 .string / .data，没有 .text
                case .string(let text):
                    self?.handleMessage(text)
                case .data(let data):
                    if let text = String(data: data, encoding: .utf8) {
                        self?.handleMessage(text)
                    }
                @unknown default:
                    break
                }
                self?.receive()
            case .failure:
                self?.isConnected = false
                // 自动重连
                DispatchQueue.global().asyncAfter(deadline: .now() + 3) {
                    // self?.reconnect()
                }
            }
        }
    }

    private func handleMessage(_ text: String) {
        guard let data = text.data(using: .utf8),
              let msg = try? JSONDecoder().decode(WSIncomingMessage.self, from: data) else { return }

        DispatchQueue.main.async {
            switch msg.type {
            case "chat":
                let bubble = ChatBubbleMessage(
                    id: msg.msgID ?? Int64(Date().timeIntervalSince1970),
                    fromUserID: msg.from ?? "",
                    toUserID: msg.to ?? "",
                    msgType: msg.msgType ?? 1,
                    content: msg.content ?? "",
                    isBurn: false,
                    burnDuration: 0,
                    timestamp: msg.timestamp ?? Int64(Date().timeIntervalSince1970)
                )
                self.messages.append(bubble)
            case "pong":
                break
            case "ack":
                break
            case "typing":
                self.typingUser = msg.from
            case "read":
                break
            default:
                break
            }
        }
    }

    private var pingTimer: Timer?
    private func startPing() {
        pingTimer = Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { _ in
            self.send(message: WSOutgoingMessage(type: "ping"))
        }
    }
}

// MARK: - WS 消息模型

struct WSOutgoingMessage: Codable {
    let type: String
    let to: String?
    let msgType: Int8?
    let content: String?

    enum CodingKeys: String, CodingKey {
        case type, to
        case msgType = "msg_type"
        case content
    }

    init(type: String, to: String? = nil, msgType: Int8? = nil, content: String? = nil) {
        self.type = type
        self.to = to
        self.msgType = msgType
        self.content = content
    }
}

struct WSIncomingMessage: Codable {
    let type: String
    let from: String?
    let to: String?
    let msgType: Int8?
    let content: String?
    let msgID: Int64?
    let timestamp: Int64?

    enum CodingKeys: String, CodingKey {
        case type, from, to
        case msgType = "msg_type"
        case content
        case msgID = "msg_id"
        case timestamp
    }
}

// MARK: - 聊天详情页

struct ChatRoomView: View {
    let friendID: String
    let friendName: String

    @StateObject private var wsManager = WebSocketManager()
    @State private var inputText = ""
    @State private var showMoreMenu = false

    var body: some View {
        VStack(spacing: 0) {
            // 连接状态指示
            if !wsManager.isConnected {
                HStack {
                    ProgressView().scaleEffect(0.7)
                    Text("连接中...")
                        .font(.system(size: 12))
                        .foregroundColor(Color("neutral-400"))
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 4)
                .background(Color("neutral-50"))
            }

            // 消息列表
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 8) {
                        ForEach(wsManager.messages) { msg in
                            MessageBubble(message: msg, onRetry: retryMessage)
                                .id(msg.id)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.top, 12)
                }
                .onChange(of: wsManager.messages.count) { _, _ in
                    if let lastID = wsManager.messages.last?.id {
                        withAnimation {
                            proxy.scrollTo(lastID, anchor: .bottom)
                        }
                    }
                }
            }

            // 输入栏
            InputBar(
                text: $inputText,
                onSend: sendMessage,
                onMore: { showMoreMenu = true }
            )
        }
        .navigationTitle(friendName)
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            if let token = APIClient.shared.accessToken,
               let userID = UserDefaults.standard.string(forKey: "user_id") {
                wsManager.connect(token: token, userID: userID)
            }
            // 加载历史消息
            loadHistory()
        }
        .onDisappear {
            wsManager.disconnect()
        }
        .sheet(isPresented: $showMoreMenu) {
            MoreMenuSheet(friendID: friendID)
        }
    }

    private func sendMessage() {
        guard !inputText.trimmingCharacters(in: .whitespaces).isEmpty else { return }

        let content = inputText
        inputText = ""

        // 先添加到本地消息列表 (sending 状态)
        let localMsg = WebSocketManager.ChatBubbleMessage(
            id: Int64(Date().timeIntervalSince1970 * 1000),
            fromUserID: WebSocketManager.ChatBubbleMessage.currentUserID,
            toUserID: friendID,
            msgType: 1,
            content: content,
            isBurn: false,
            burnDuration: 0,
            timestamp: Int64(Date().timeIntervalSince1970),
            sendStatus: .sending
        )
        wsManager.messages.append(localMsg)

        // WebSocket 发送
        wsManager.send(message: WSOutgoingMessage(
            type: "chat",
            to: friendID,
            msgType: 1,
            content: content
        ))

        // HTTP API 发送（确保持久化）
        Task {
            struct SendResult: Codable { let msg_id: Int64; let timestamp: Int64 }
            do {
                let _: SendResult = try await APIClient.shared.post("/messages/send", body: [
                    "to_user_id": friendID,
                    "msg_type": 1,
                    "content": content
                ])
                // 发送成功 → 更新状态
                await MainActor.run {
                    if let idx = wsManager.messages.firstIndex(where: { $0.id == localMsg.id }) {
                        wsManager.messages[idx].sendStatus = .sent
                    }
                }
            } catch {
                // 发送失败 → 标记 failed
                await MainActor.run {
                    if let idx = wsManager.messages.firstIndex(where: { $0.id == localMsg.id }) {
                        wsManager.messages[idx].sendStatus = .failed
                    }
                }
            }
        }
    }

    /// BUG-003 修复: 重试发送失败的消息
    private func retryMessage(_ msg: WebSocketManager.ChatBubbleMessage) {
        // 先移除旧消息
        wsManager.messages.removeAll { $0.id == msg.id }
        // 重新发送
        inputText = msg.content
        sendMessage()
    }

    private func loadHistory() {
        Task {
            struct HistoryResult: Codable {
                let messages: [HistoryMessage]
            }
            struct HistoryMessage: Codable {
                let id: Int64
                let from_user_id: String
                let to_user_id: String
                let msg_type: Int8
                let content: String
                let is_burn: Int8
                let burn_duration: Int
                let created_at: Int64
            }

            do {
                let result: HistoryResult = try await APIClient.shared.get(
                    "/messages/history",
                    query: ["friend_id": friendID, "page": "1", "size": "50"]
                )
                await MainActor.run {
                    wsManager.messages = result.messages.map { m in
                        WebSocketManager.ChatBubbleMessage(
                            id: m.id,
                            fromUserID: m.from_user_id,
                            toUserID: m.to_user_id,
                            msgType: m.msg_type,
                            content: m.content,
                            isBurn: m.is_burn == 1,
                            burnDuration: m.burn_duration,
                            timestamp: m.created_at
                        )
                    }
                }
            } catch {
                // 静默失败
            }
        }
    }
}

// MARK: - 消息气泡

struct MessageBubble: View {
    let message: WebSocketManager.ChatBubbleMessage
    var onRetry: ((WebSocketManager.ChatBubbleMessage) -> Void)?

    var body: some View {
        HStack {
            if message.isMe { Spacer() }

            VStack(alignment: message.isMe ? .trailing : .leading, spacing: 4) {
                // 气泡
                Text(message.content)
                    .font(.system(size: 15))
                    .foregroundColor(message.isBurn ? .white : (message.isMe ? .white : Color("neutral-800")))
                    .padding(.horizontal, 12)
                    .padding(.vertical, 10)
                    .background(
                        message.isBurn
                            ? AnyShapeStyle(Color("neutral-900"))
                            : (message.isMe
                                ? AnyShapeStyle(LinearGradient(
                                    colors: [Color(hex: "FF6B4A"), Color(hex: "FF3B30")],
                                    startPoint: .leading, endPoint: .trailing))
                                : AnyShapeStyle(Color("neutral-100")))
                    )
                    .cornerRadius(12)
                    .opacity(message.sendStatus == .sending ? 0.5 : 1.0)

                // 时间戳 + 发送状态
                HStack(spacing: 4) {
                    if message.sendStatus == .sending {
                        ProgressView()
                            .scaleEffect(0.6)
                            .frame(width: 12, height: 12)
                    }
                    if message.sendStatus == .failed {
                        Button(action: { onRetry?(message) }) {
                            Image(systemName: "exclamationmark.circle.fill")
                                .font(.system(size: 14))
                                .foregroundColor(.red)
                        }
                    }
                    Text(formatTime(message.timestamp))
                        .font(.system(size: 11))
                        .foregroundColor(Color("neutral-400"))
                }
            }
            .frame(maxWidth: UIScreen.main.bounds.width * 0.7, alignment: message.isMe ? .trailing : .leading)

            if !message.isMe { Spacer() }
        }
    }

    private func formatTime(_ ts: Int64) -> String {
        let date = Date(timeIntervalSince1970: TimeInterval(ts))
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }
}

// MARK: - 输入栏

struct InputBar: View {
    @Binding var text: String
    let onSend: () -> Void
    let onMore: () -> Void

    var body: some View {
        HStack(spacing: 8) {
            // 输入框
            TextField("输入消息...", text: $text, axis: .vertical)
                .font(.system(size: 15))
                .lineLimit(1...4)
                .padding(.horizontal, 16)
                .padding(.vertical, 8)
                .background(Color("neutral-50"))
                .cornerRadius(12)

            // 发送/更多按钮
            if text.trimmingCharacters(in: .whitespaces).isEmpty {
                Button(action: onMore) {
                    Image(systemName: "plus.circle.fill")
                        .font(.system(size: 24))
                        .foregroundColor(Color("neutral-400"))
                }
            } else {
                Button(action: onSend) {
                    Image(systemName: "paperplane.fill")
                        .font(.system(size: 16))
                        .foregroundColor(.white)
                        .frame(width: 32, height: 32)
                        .background(Color(hex: "FF6B4A"))
                        .clipShape(Circle())
                }
                .transition(.scale.combined(with: .opacity))
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
        .animation(.easeInOut(duration: 0.2), value: text.isEmpty)
    }
}

// MARK: - 更多菜单

struct MoreMenuSheet: View {
    let friendID: String

    var body: some View {
        VStack(spacing: 0) {
            // 顶部拖拽条
            RoundedRectangle(cornerRadius: 3)
                .fill(Color("neutral-200"))
                .frame(width: 36, height: 5)
                .padding(.top, 8)

            HStack(spacing: 32) {
                MenuButton(icon: "photo", title: "相册", color: Color(hex: "FF6B4A"))
                MenuButton(icon: "camera", title: "拍照", color: Color(hex: "7C5CFC"))
                MenuButton(icon: "flame", title: "阅后即焚", color: Color(hex: "FF3B30"))
            }
            .padding(.vertical, 32)

            Spacer()
        }
        .presentationDetents([.height(200)])
        .presentationDragIndicator(.visible)
    }
}

struct MenuButton: View {
    let icon: String
    let title: String
    let color: Color

    var body: some View {
        VStack(spacing: 8) {
            Circle()
                .fill(color.opacity(0.15))
                .frame(width: 56, height: 56)
                .overlay(
                    Image(systemName: icon)
                        .font(.system(size: 22))
                        .foregroundColor(color)
                )
            Text(title)
                .font(.system(size: 13))
                .foregroundColor(Color("neutral-600"))
        }
    }
}
