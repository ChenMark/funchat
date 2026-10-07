// FunChat iOS - 解锁条件 UI (COND-004/006 + LOC-001 + QUIZ-001)
// 条件设置入口 + 锁定消息卡片 + 地图选点 + 答题
// 基于 DES-003 设计稿 P5/P6

import SwiftUI

// MARK: - 条件设置入口 (COND-004)

struct ConditionSettingsEntryView: View {
    let friendID: String
    @State private var conditions: [ConditionItem] = []
    @State private var showLocationPicker = false
    @State private var showQuizSetup = false
    @State private var isLoading = false

    var body: some View {
        VStack(spacing: 0) {
            Text("选择对方需要满足的条件才能查看你的消息")
                .font(.system(size: 15))
                .foregroundColor(Color("neutral-600"))
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 16)
                .padding(.top, 24)

            VStack(spacing: 16) {
                // 定位解锁
                ConditionCard(
                    icon: "location.fill",
                    title: "定位解锁",
                    subtitle: "需要在一定范围内",
                    color: Color(hex: "20C0B0"),
                    isEnabled: conditions.contains { $0.type == 1 && $0.enabled }
                ) {
                    showLocationPicker = true
                }

                // 步数解锁
                ConditionCard(
                    icon: "figure.walk",
                    title: "步数解锁",
                    subtitle: "需要达到目标步数",
                    color: Color(hex: "FFB020"),
                    isEnabled: conditions.contains { $0.type == 2 && $0.enabled }
                ) {
                    // 步数选择器
                }

                // 答题解锁
                ConditionCard(
                    icon: "brain.head.profile",
                    title: "答题解锁",
                    subtitle: "需要答对你的问题",
                    color: Color(hex: "7C5CFC"),
                    isEnabled: conditions.contains { $0.type == 3 && $0.enabled }
                ) {
                    showQuizSetup = true
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 24)

            Text("可同时开启多个条件，满足任一即可解锁")
                .font(.system(size: 11))
                .foregroundColor(Color("neutral-400"))
                .padding(.top, 16)

            Spacer()

            // 保存按钮
            Button(action: saveConditions) {
                Text("保存条件")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 48)
                    .background(LinearGradient(
                        colors: [Color(hex: "FF6B4A"), Color(hex: "FF3B30")],
                        startPoint: .leading, endPoint: .trailing))
                    .cornerRadius(12)
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 32)
        }
        .navigationTitle("设置解锁条件")
        .navigationBarTitleDisplayMode(.inline)
        .task { await loadConditions() }
        .sheet(isPresented: $showLocationPicker) {
            LocationPickerView(friendID: friendID)
        }
        .sheet(isPresented: $showQuizSetup) {
            QuizSetupView()
        }
    }

    private func loadConditions() async {
        struct Wrapper: Codable { let conditions: [ConditionItem] }
        do {
            let result: Wrapper = try await APIClient.shared.get("/conditions/\(friendID)")
            conditions = result.conditions
        } catch {
            conditions = []
        }
    }

    private func saveConditions() {
        // TODO: 调用 POST /conditions/set
    }
}

struct ConditionItem: Codable {
    let id: Int64
    let condType: Int8
    let label: String
    let isEnabled: Int8
    let params: String
    var type: Int8 { condType }
    var enabled: Bool { isEnabled == 1 }

    enum CodingKeys: String, CodingKey {
        case id
        case condType = "cond_type"
        case label
        case isEnabled = "is_enabled"
        case params
    }
}

struct ConditionCard: View {
    let icon: String
    let title: String
    let subtitle: String
    let color: Color
    let isEnabled: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 12) {
                ZStack {
                    Circle()
                        .fill(color.opacity(0.15))
                        .frame(width: 44, height: 44)
                    Image(systemName: icon)
                        .font(.system(size: 20))
                        .foregroundColor(color)
                }
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(Color("neutral-800"))
                    Text(subtitle)
                        .font(.system(size: 13))
                        .foregroundColor(Color("neutral-600"))
                }
                Spacer()
                Toggle("", isOn: .constant(isEnabled))
                    .tint(color)
                    .labelsHidden()
            }
            .padding(16)
            .background(Color("neutral-0"))
            .cornerRadius(12)
            .shadow(color: Color.black.opacity(0.04), radius: 8, y: 2)
        }
        .buttonStyle(PlainButtonStyle())
    }
}

// MARK: - 锁定消息卡片 (COND-006)

struct LockedMessageCard: View {
    let conditions: [CondStatusItem]
    let onUnlock: () -> Void

    var body: some View {
        VStack(spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "lock.fill")
                    .foregroundColor(Color(hex: "FF9500"))
                Text("对方设置了聊天条件")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundColor(Color(hex: "FF9500"))
            }

            ForEach(conditions) { cond in
                HStack(spacing: 8) {
                    Circle()
                        .fill(cond.met ? Color(hex: "34C759") : Color(hex: "FF9500"))
                        .frame(width: 8, height: 8)
                    Image(systemName: cond.icon)
                        .font(.system(size: 14))
                        .foregroundColor(Color("neutral-600"))
                    Text(cond.label)
                        .font(.system(size: 14))
                        .foregroundColor(Color("neutral-800"))
                    Spacer()
                    Text(cond.met ? "已满足" : "未满足")
                        .font(.system(size: 12))
                        .foregroundColor(cond.met ? Color(hex: "34C759") : Color("neutral-400"))
                }
            }

            Text("满足任一条件即可解锁")
                .font(.system(size: 11))
                .foregroundColor(Color("neutral-400"))

            Button(action: onUnlock) {
                Text("开始解锁")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundColor(.white)
                    .frame(width: 140, height: 36)
                    .background(LinearGradient(
                        colors: [Color(hex: "FF6B4A"), Color(hex: "FF3B30")],
                        startPoint: .leading, endPoint: .trailing))
                    .cornerRadius(8)
            }
        }
        .padding(16)
        .background(Color(hex: "FFF8E1"))
        .cornerRadius(8)
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color(hex: "FF9500").opacity(0.3), lineWidth: 1)
        )
    }
}

struct CondStatusItem: Identifiable {
    let id = UUID()
    let type: Int8
    let label: String
    let icon: String
    let met: Bool
}

// MARK: - 地图选点页 (LOC-001)

struct LocationPickerView: View {
    let friendID: String
    @State private var radius: Double = 500
    @State private var address = "正在获取位置..."
    @Environment(\.dismiss) var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // 地图占位（实际接入高德 SDK MapView）
                ZStack {
                    Rectangle()
                        .fill(Color("neutral-100"))
                    VStack(spacing: 8) {
                        Image(systemName: "map.fill")
                            .font(.system(size: 48))
                            .foregroundColor(Color(hex: "20C0B0"))
                        Text("高德地图选点")
                            .font(.system(size: 15))
                            .foregroundColor(Color("neutral-600"))
                    }
                }
                .frame(height: 300)

                // 距离滑块
                VStack(spacing: 12) {
                    Text("需要相距在 \(Int(radius))m 内")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(Color("neutral-800"))

                    Slider(value: $radius, in: 200...2000, step: 100)
                        .tint(Color(hex: "FF6B4A"))

                    HStack {
                        Text("200m").font(.system(size: 11)).foregroundColor(Color("neutral-400"))
                        Spacer()
                        Text("2km").font(.system(size: 11)).foregroundColor(Color("neutral-400"))
                    }
                }
                .padding(16)

                Spacer()

                Button(action: {
                    // 保存条件
                    dismiss()
                }) {
                    Text("确定")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(.white)
                        .frame(maxWidth: .infinity)
                        .frame(height: 48)
                        .background(Color(hex: "FF6B4A"))
                        .cornerRadius(12)
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 32)
            }
            .navigationTitle("选择位置范围")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("取消") { dismiss() }
                }
            }
        }
    }
}

// MARK: - 答题设置页 (QUIZ-001)

struct QuizSetupView: View {
    @State private var question = ""
    @State private var answer = ""
    @State private var maxTries = 3
    @Environment(\.dismiss) var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("输入你的问题")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundColor(Color("neutral-600"))
                    TextField("例如：我最喜欢的颜色是什么？", text: $question)
                        .font(.system(size: 15))
                        .padding(16)
                        .background(Color("neutral-50"))
                        .cornerRadius(8)
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text("输入正确答案")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundColor(Color("neutral-600"))
                    TextField("答案", text: $answer)
                        .font(.system(size: 15))
                        .padding(16)
                        .background(Color("neutral-50"))
                        .cornerRadius(8)
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text("允许尝试次数")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundColor(Color("neutral-600"))
                    Picker("", selection: $maxTries) {
                        Text("1次").tag(1)
                        Text("2次").tag(2)
                        Text("3次").tag(3)
                        Text("5次").tag(5)
                    }
                    .pickerStyle(.segmented)
                }

                Spacer()

                Button(action: {
                    Task {
                        struct QuizResult: Codable { let quiz_id: Int64 }
                        let _: QuizResult? = try? await APIClient.shared.post("/conditions/set-quiz", body: [
                            "question": question,
                            "answer": answer,
                            "max_tries": maxTries
                        ])
                        dismiss()
                    }
                }) {
                    Text("保存题目")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(.white)
                        .frame(maxWidth: .infinity)
                        .frame(height: 48)
                        .background(canSave
                            ? AnyShapeStyle(LinearGradient(
                                colors: [Color(hex: "FF6B4A"), Color(hex: "FF3B30")],
                                startPoint: .leading, endPoint: .trailing))
                            : AnyShapeStyle(Color("neutral-100")))
                        .cornerRadius(12)
                }
                .disabled(!canSave)
                .padding(.horizontal, 16)
                .padding(.bottom, 32)
            }
            .padding(.horizontal, 16)
            .padding(.top, 24)
            .navigationTitle("设置答题条件")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("取消") { dismiss() }
                }
            }
        }
    }

    private var canSave: Bool {
        !question.trimmingCharacters(in: .whitespaces).isEmpty &&
        !answer.trimmingCharacters(in: .whitespaces).isEmpty
    }
}

// MARK: - 解锁验证弹窗 (LOC-004 验证中)

struct UnlockVerificationSheet: View {
    let condType: Int8 // 1=定位 2=步数 3=答题
    @State private var isVerifying = true
    @State private var success = false
    @State private var message = "正在验证..."

    var body: some View {
        VStack(spacing: 16) {
            RoundedRectangle(cornerRadius: 3)
                .fill(Color("neutral-200"))
                .frame(width: 36, height: 5)
                .padding(.top, 8)

            Text("🔒 解锁聊天")
                .font(.system(size: 18, weight: .semibold))
                .foregroundColor(Color("neutral-800"))

            // 动画图标
            ZStack {
                Circle()
                    .fill(Color("neutral-50"))
                    .frame(width: 80, height: 80)
                if isVerifying {
                    ProgressView()
                        .scaleEffect(1.5)
                        .tint(Color(hex: "FF6B4A"))
                } else if success {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.system(size: 40))
                        .foregroundColor(Color(hex: "34C759"))
                } else {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 40))
                        .foregroundColor(Color(hex: "FF3B30"))
                }
            }

            Text(message)
                .font(.system(size: 15))
                .foregroundColor(Color("neutral-600"))

            if isVerifying {
                // 进度条
                ProgressView(value: 0.6)
                    .tint(Color(hex: "FF6B4A"))
                    .padding(.horizontal, 32)
            }

            if !isVerifying {
                Button(action: { /* dismiss */ }) {
                    Text(success ? "开始聊天" : "重试")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(.white)
                        .frame(maxWidth: .infinity)
                        .frame(height: 44)
                        .background(success ? Color(hex: "FF6B4A") : Color(hex: "FF9500"))
                        .cornerRadius(12)
                }
                .padding(.horizontal, 16)
            }

            Spacer()
        }
        .presentationDetents([.height(360)])
        .onAppear {
            // 模拟验证
            DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                isVerifying = false
                success = true
                message = "验证成功！"
            }
        }
    }
}
