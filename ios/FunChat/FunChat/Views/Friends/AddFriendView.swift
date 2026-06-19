// FunChat iOS - 添加好友页 (FRD-006)
// 基于 DES-003 设计稿 P4

import SwiftUI

struct AddFriendView: View {
    @State private var searchText = ""
    @State private var searchResults: [SearchUser] = []
    @State private var isSearching = false
    @State private var requests: [FriendRequestItem] = []
    @State private var showScanner = false
    @State private var toastMessage = ""
    @State private var showToast = false

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                // 搜索栏
                HStack(spacing: 8) {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(Color("neutral-400"))
                        .font(.system(size: 16))
                    TextField("搜索手机号或用户名", text: $searchText)
                        .font(.system(size: 15))
                        .onChange(of: searchText) { _, newValue in
                            if newValue.count >= 2 {
                                search(keyword: newValue)
                            } else {
                                searchResults = []
                            }
                        }
                }
                .padding(.horizontal, 16)
                .frame(height: 40)
                .background(Color("neutral-50"))
                .cornerRadius(12)
                .padding(.horizontal, 16)

                // 搜索结果
                if !searchResults.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("搜索结果")
                            .font(.system(size: 11))
                            .foregroundColor(Color("neutral-400"))
                            .padding(.horizontal, 16)

                        ForEach(searchResults) { user in
                            HStack(spacing: 12) {
                                Circle()
                                    .fill(Color(hex: "FFD9D1"))
                                    .frame(width: 36, height: 36)
                                    .overlay(
                                        Text(String(user.nickname.prefix(1)))
                                            .font(.system(size: 14, weight: .medium))
                                            .foregroundColor(Color(hex: "FF6B4A"))
                                    )
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(user.nickname)
                                        .font(.system(size: 16, weight: .medium))
                                        .foregroundColor(Color("neutral-800"))
                                    Text("ID: \(user.userId)")
                                        .font(.system(size: 11))
                                        .foregroundColor(Color("neutral-400"))
                                }
                                Spacer()
                                if user.isFriend {
                                    Text("已添加")
                                        .font(.system(size: 13))
                                        .foregroundColor(Color("neutral-400"))
                                } else {
                                    Button("添加") {
                                        sendRequest(to: user.userId)
                                    }
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundColor(Color(hex: "FF6B4A"))
                                    .padding(.horizontal, 16)
                                    .padding(.vertical, 6)
                                    .background(Color(hex: "FFD9D1"))
                                    .cornerRadius(8)
                                }
                            }
                            .padding(.horizontal, 16)
                        }
                    }
                } else if isSearching {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding()
                }

                // 扫码添加
                Button(action: { showScanner = true }) {
                    HStack(spacing: 12) {
                        Image(systemName: "qrcode.viewfinder")
                            .font(.system(size: 24))
                            .foregroundColor(Color(hex: "FF6B4A"))
                        Text("扫一扫")
                            .font(.system(size: 16, weight: .medium))
                            .foregroundColor(Color("neutral-800"))
                        Spacer()
                    }
                    .padding(16)
                    .background(Color("neutral-0"))
                    .cornerRadius(12)
                    .shadow(color: Color.black.opacity(0.04), radius: 4)
                }
                .padding(.horizontal, 16)

                // 好友请求
                if !requests.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("好友请求")
                            .font(.system(size: 11))
                            .foregroundColor(Color("neutral-400"))
                            .padding(.horizontal, 16)

                        ForEach(requests) { req in
                            HStack(spacing: 12) {
                                Circle()
                                    .fill(Color(hex: "FFD9D1"))
                                    .frame(width: 36, height: 36)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(req.fromUserId)
                                        .font(.system(size: 16, weight: .medium))
                                        .foregroundColor(Color("neutral-800"))
                                    if !req.message.isEmpty {
                                        Text(req.message)
                                            .font(.system(size: 13))
                                            .foregroundColor(Color("neutral-600"))
                                            .lineLimit(1)
                                    }
                                }
                                Spacer()
                                // 接受/拒绝
                                Button("✓") {
                                    acceptRequest(req.id)
                                }
                                .font(.system(size: 18, weight: .bold))
                                .foregroundColor(.green)
                                .frame(width: 32, height: 32)
                                .background(Color.green.opacity(0.1))
                                .clipShape(Circle())

                                Button("✕") {
                                    rejectRequest(req.id)
                                }
                                .font(.system(size: 18, weight: .bold))
                                .foregroundColor(.red)
                                .frame(width: 32, height: 32)
                                .background(Color.red.opacity(0.1))
                                .clipShape(Circle())
                            }
                            .padding(.horizontal, 16)
                        }
                    }
                }
            }
            .padding(.vertical, 16)
        }
        .navigationTitle("添加好友")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await loadRequests()
        }
        .overlay {
            if showToast {
                ToastView(message: toastMessage)
                    .transition(.opacity)
            }
        }
    }

    // MARK: - API 调用

    private func search(keyword: String) {
        isSearching = true
        Task {
            do {
                let result = try await APIClient.shared.searchFriends(keyword: keyword)
                await MainActor.run {
                    searchResults = result.users
                    isSearching = false
                }
            } catch {
                await MainActor.run {
                    isSearching = false
                    searchResults = []
                }
            }
        }
    }

    private func loadRequests() async {
        do {
            requests = try await APIClient.shared.getFriendRequests()
        } catch {
            requests = []
        }
    }

    private func sendRequest(to userId: String) {
        Task {
            do {
                _ = try await APIClient.shared.sendFriendRequest(targetUserId: userId, message: nil)
                await MainActor.run {
                    toastMessage = "申请已发送"
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) { showToast = false }
                }
            } catch let error as APIError {
                await MainActor.run {
                    toastMessage = error.message
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) { showToast = false }
                }
            } catch {
                await MainActor.run {
                    toastMessage = "网络异常"
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) { showToast = false }
                }
            }
        }
    }

    private func acceptRequest(_ id: Int64) {
        Task {
            do {
                _ = try await APIClient.shared.acceptFriendRequest(requestId: id)
                await MainActor.run {
                    requests.removeAll { $0.id == id }
                    toastMessage = "已添加为好友"
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) { showToast = false }
                }
            } catch {
                await MainActor.run {
                    toastMessage = "操作失败"
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) { showToast = false }
                }
            }
        }
    }

    private func rejectRequest(_ id: Int64) {
        Task {
            do {
                _ = try await APIClient.shared.rejectFriendRequest(requestId: id)
                await MainActor.run {
                    requests.removeAll { $0.id == id }
                }
            } catch {
                await MainActor.run {
                    toastMessage = "操作失败"
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) { showToast = false }
                }
            }
        }
    }
}
