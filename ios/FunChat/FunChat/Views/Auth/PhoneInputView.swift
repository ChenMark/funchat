// FunChat iOS - 手机号输入页 (AUTH-005)
// 基于 DES-002 设计稿 P3

import SwiftUI

struct PhoneInputView: View {
    @State private var phone = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var agreedToTerms = false
    @State private var navigateToCode = false
    @State private var codeToken = ""

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // 页面标题
                VStack(alignment: .leading, spacing: 8) {
                    Text("输入手机号")
                        .font(.system(size: 28, weight: .bold))
                        .foregroundColor(Color("neutral-900"))
                    Text("使用短信验证码登录")
                        .font(.system(size: 15))
                        .foregroundColor(Color("neutral-600"))
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 16)
                .padding(.top, 32)

                // 手机号输入区
                HStack(spacing: 12) {
                    // 区号
                    HStack(spacing: 4) {
                        Text("🇨🇳 +86")
                            .font(.system(size: 15, weight: .medium))
                        Image(systemName: "chevron.down")
                            .font(.system(size: 12))
                            .foregroundColor(Color("neutral-400"))
                    }
                    // 分隔线
                    Rectangle()
                        .fill(Color("neutral-200"))
                        .frame(width: 1, height: 20)
                    // 手机号输入
                    TextField("请输入手机号", text: $phone)
                        .font(.system(size: 15))
                        .foregroundColor(Color("neutral-800"))
                        .keyboardType(.numberPad)
                        .onChange(of: phone) { _, newValue in
                            // 仅保留数字，最多11位
                            let filtered = newValue.filter { $0.isNumber }
                            phone = String(filtered.prefix(11))
                            // 格式化: 138 0000 0000
                            if phone.count > 3 && phone.count <= 7 {
                                phone.insert(" ", at: phone.index(phone.startIndex, offsetBy: 3))
                            } else if phone.count > 7 {
                                phone.insert(" ", at: phone.index(phone.startIndex, offsetBy: 3))
                                phone.insert(" ", at: phone.index(phone.startIndex, offsetBy: 8))
                            }
                            showError = false
                        }
                    Spacer()
                }
                .padding(.horizontal, 16)
                .frame(height: 56)
                .background(Color("neutral-50"))
                .cornerRadius(8)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(showError ? Color.red : Color("neutral-200"), lineWidth: showError ? 2 : 1)
                )
                .padding(.horizontal, 16)
                .padding(.top, 24)

                // 错误提示
                if showError {
                    Text(errorMessage)
                        .font(.system(size: 13))
                        .foregroundColor(.red)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 16)
                        .padding(.top, 8)
                }

                Spacer()

                // 获取验证码按钮
                Button(action: sendCode) {
                    HStack {
                        if isLoading {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Text("获取验证码")
                                .font(.system(size: 16, weight: .semibold))
                                .foregroundColor(canSend ? .white : Color("neutral-400"))
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .frame(height: 48)
                    .background(canSend ? AnyShapeStyle(LinearGradient(
                        colors: [Color(hex: "FF6B4A"), Color(hex: "FF3B30")],
                        startPoint: .leading, endPoint: .trailing
                    )) : AnyShapeStyle(Color("neutral-100")))
                    .cornerRadius(12)
                }
                .disabled(!canSend || isLoading)
                .padding(.horizontal, 16)

                // 协议勾选
                HStack(spacing: 6) {
                    Button(action: { agreedToTerms.toggle() }) {
                        Image(systemName: agreedToTerms ? "checkmark.circle.fill" : "circle")
                            .foregroundColor(agreedToTerms ? Color(hex: "FF6B4A") : Color("neutral-200"))
                            .font(.system(size: 16))
                    }
                    Text("我已阅读并同意")
                        .font(.system(size: 11))
                        .foregroundColor(Color("neutral-600"))
                    Text("《用户协议》")
                        .font(.system(size: 11))
                        .foregroundColor(Color(hex: "007AFF"))
                        .underline()
                    Text("和")
                        .font(.system(size: 11))
                        .foregroundColor(Color("neutral-600"))
                    Text("《隐私政策》")
                        .font(.system(size: 11))
                        .foregroundColor(Color(hex: "007AFF"))
                        .underline()
                }
                .padding(.top, 16)
                .padding(.bottom, 32)
            }
            .navigationTitle("登录")
            .navigationBarTitleDisplayMode(.inline)
            .navigationDestination(isPresented: $navigateToCode) {
                CodeInputView(phone: phone, codeToken: codeToken)
            }
        }
    }

    private var canSend: Bool {
        let digits = phone.filter { $0.isNumber }
        return digits.count == 11 && agreedToTerms && !isLoading
    }

    private func sendCode() {
        isLoading = true
        showError = false
        let digits = phone.filter { $0.isNumber }

        Task {
            do {
                let result = try await APIClient.shared.sendCode(phone: digits)
                await MainActor.run {
                    isLoading = false
                    #if DEBUG
                    // 开发环境自动填充验证码
                    codeToken = result.code ?? ""
                    #endif
                    navigateToCode = true
                }
            } catch let error as APIError {
                await MainActor.run {
                    isLoading = false
                    errorMessage = error.message
                    showError = true
                }
            } catch {
                await MainActor.run {
                    isLoading = false
                    errorMessage = "网络异常，请检查网络"
                    showError = true
                }
            }
        }
    }
}

// FunChat iOS - 验证码输入页 (AUTH-005)
// 基于 DES-002 设计稿 P4

import SwiftUI

struct CodeInputView: View {
    let phone: String
    @State var codeToken: String

    @State private var code = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var countdown = 60
    @State private var canResend = false
    @State private var navigateToProfile = false
    @State private var navigateToHome = false
    @State private var isNewUser = false
    @State private var timer: Timer?

    var body: some View {
        VStack(spacing: 0) {
            // 标题
            VStack(alignment: .leading, spacing: 8) {
                Text("输入验证码")
                    .font(.system(size: 28, weight: .bold))
                    .foregroundColor(Color("neutral-900"))
                Text("已发送至 +86 \(maskPhone(phone))")
                    .font(.system(size: 15))
                    .foregroundColor(Color("neutral-600"))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 16)
            .padding(.top, 32)

            // 6 格验证码输入
            HStack(spacing: 8) {
                ForEach(0..<6, id: \.self) { index in
                    ZStack {
                        RoundedRectangle(cornerRadius: 8)
                            .fill(Color("neutral-50"))
                            .overlay(
                                RoundedRectangle(cornerRadius: 8)
                                    .stroke(borderColor(for: index), lineWidth: borderWidth(for: index))
                            )
                        if index < code.count {
                            Text(String(code[code.index(code.startIndex, offsetBy: index)]))
                                .font(.system(size: 22, weight: .semibold))
                                .foregroundColor(Color("neutral-800"))
                        }
                        if index == code.count {
                            // 光标
                            Rectangle()
                                .fill(Color(hex: "FF6B4A"))
                                .frame(width: 2, height: 20)
                                .opacity(showCursor ? 1 : 0)
                        }
                    }
                    .frame(width: 48, height: 48)
                }
            }
            .padding(.top, 32)

            // 错误提示
            if showError {
                Text(errorMessage)
                    .font(.system(size: 13))
                    .foregroundColor(.red)
                    .padding(.top, 8)
            }

            // 倒计时 / 重新发送
            Button(action: resendCode) {
                if canResend {
                    Text("重新发送")
                        .font(.system(size: 13))
                        .foregroundColor(Color(hex: "007AFF"))
                        .underline()
                } else {
                    Text("\(countdown)s 后重新发送")
                        .font(.system(size: 13))
                        .foregroundColor(Color("neutral-600"))
                }
            }
            .disabled(!canResend)
            .padding(.top, 16)

            Spacer()

            // 登录按钮
            Button(action: verifyAndLogin) {
                HStack {
                    if isLoading {
                        ProgressView().tint(.white)
                    } else {
                        Text(isNewUser ? "注册" : "登录")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundColor(code.count == 6 ? .white : Color("neutral-400"))
                    }
                }
                .frame(maxWidth: .infinity)
                .frame(height: 48)
                .background(code.count == 6 ? AnyShapeStyle(LinearGradient(
                    colors: [Color(hex: "FF6B4A"), Color(hex: "FF3B30")],
                    startPoint: .leading, endPoint: .trailing
                )) : AnyShapeStyle(Color("neutral-100")))
                .cornerRadius(12)
            }
            .disabled(code.count != 6 || isLoading)
            .padding(.horizontal, 16)
            .padding(.bottom, 32)
        }
        .navigationTitle("验证码")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            startCountdown()
            // 自动聚焦输入
            // iOS 15+ 使用 focusState
        }
        .onDisappear { timer?.invalidate() }
        .background(
            // 隐藏的 TextField 用于接收键盘输入
            TextField("", text: $code)
                .keyboardType(.numberPad)
                .opacity(0)
                .frame(width: 0, height: 0)
                .onChange(of: code) { _, newValue in
                    code = String(newValue.filter { $0.isNumber }.prefix(6))
                    showError = false
                    if code.count == 6 {
                        verifyAndLogin()
                    }
                }
        )
    }

    @State private var showCursor = true

    // MARK: - 辅助方法

    private func maskPhone(_ phone: String) -> String {
        let digits = phone.filter { $0.isNumber }
        guard digits.count == 11 else { return phone }
        let start = digits.prefix(3)
        let end = digits.suffix(4)
        return "\(start)****\(end)"
    }

    private func borderColor(for index: Int) -> Color {
        if showError && index < 6 { return .red }
        if index < code.count { return Color(hex: "FF6B4A") }
        if index == code.count { return Color(hex: "FF6B4A") }
        return Color("neutral-200")
    }

    private func borderWidth(for index: Int) -> CGFloat {
        if index == code.count || showError { return 2 }
        if index < code.count { return 1 }
        return 1
    }

    private func startCountdown() {
        countdown = 60
        canResend = false
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { t in
            countdown -= 1
            if countdown <= 0 {
                canResend = true
                t.invalidate()
            }
        }
    }

    private func resendCode() {
        let digits = phone.filter { $0.isNumber }
        Task {
            try? await APIClient.shared.sendCode(phone: digits)
            await MainActor.run { startCountdown() }
        }
    }

    private func verifyAndLogin() {
        isLoading = true
        showError = false
        let digits = phone.filter { $0.isNumber }

        Task {
            do {
                // 1. 校验验证码
                let verifyResult = try await APIClient.shared.verifyCode(phone: digits, code: code)
                let ct = verifyResult.codeToken ?? codeToken
                isNewUser = verifyResult.isNew ?? false

                // 2. 注册或登录
                let authResult: AuthResult
                if isNewUser {
                    authResult = try await APIClient.shared.register(phone: digits, codeToken: ct, nickname: nil, avatar: nil)
                } else {
                    authResult = try await APIClient.shared.login(phone: digits, codeToken: ct)
                }

                // 3. 保存登录态
                APIClient.shared.saveAuth(authResult)

                await MainActor.run {
                    isLoading = false
                    if isNewUser {
                        navigateToProfile = true
                    } else {
                        navigateToHome = true
                    }
                }
            } catch let error as APIError {
                await MainActor.run {
                    isLoading = false
                    errorMessage = error.message
                    showError = true
                    code = ""  // 清空输入
                }
            } catch {
                await MainActor.run {
                    isLoading = false
                    errorMessage = "网络异常，请检查网络"
                    showError = true
                }
            }
        }
    }
}

// FunChat iOS - 信息完善页 (AUTH-005)
// 基于 DES-002 设计稿 P5

import SwiftUI

struct ProfileSetupView: View {
    @State private var nickname = ""
    @State private var avatarImage: UIImage?
    @State private var showImagePicker = false
    @State private var isLoading = false
    @State private var navigateToHome = false

    var body: some View {
        VStack(spacing: 0) {
            // 头像
            VStack(spacing: 8) {
                ZStack {
                    Circle()
                        .fill(Color(hex: "FFD9D1"))
                        .frame(width: 96, height: 96)
                    if let image = avatarImage {
                        Image(uiImage: image)
                            .resizable()
                            .scaledToFill()
                            .frame(width: 96, height: 96)
                            .clipShape(Circle())
                    } else {
                        Image(systemName: "plus")
                            .font(.system(size: 28, weight: .medium))
                            .foregroundColor(Color(hex: "FF6B4A"))
                    }
                }
                .onTapGesture { showImagePicker = true }
                Text("点击设置头像")
                    .font(.system(size: 11))
                    .foregroundColor(Color("neutral-400"))
            }
            .padding(.top, 32)

            // 昵称输入
            VStack(alignment: .trailing, spacing: 4) {
                TextField("输入你的昵称", text: $nickname)
                    .font(.system(size: 15))
                    .padding(.horizontal, 16)
                    .frame(height: 48)
                    .background(Color("neutral-50"))
                    .cornerRadius(8)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(Color("neutral-200"), lineWidth: 1)
                    )
                    .onChange(of: nickname) { _, newValue in
                        if newValue.count > 20 {
                            nickname = String(newValue.prefix(20))
                        }
                    }
                Text("\(nickname.count)/20")
                    .font(.system(size: 11))
                    .foregroundColor(Color("neutral-400"))
            }
            .padding(.horizontal, 16)
            .padding(.top, 32)

            Spacer()

            // 进入按钮
            Button(action: { navigateToHome = true }) {
                Text("进入 FunChat")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 48)
                    .background(LinearGradient(
                        colors: [Color(hex: "FF6B4A"), Color(hex: "FF3B30")],
                        startPoint: .leading, endPoint: .trailing
                    ))
                    .cornerRadius(12)
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 32)
        }
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button("跳过") { navigateToHome = true }
                    .foregroundColor(Color(hex: "007AFF"))
            }
        }
        .navigationDestination(isPresented: $navigateToHome) {
            // HomeView()
            Text("首页").font(.largeTitle)
        }
        .sheet(isPresented: $showImagePicker) {
            // ImagePicker
            Text("选择头像")
        }
    }
}

// MARK: - Color Extension

extension Color {
    init(hex: String) {
        let scanner = Scanner(string: hex)
        var rgb: UInt64 = 0
        scanner.scanHexInt64(&rgb)
        self.init(
            red: Double((rgb >> 16) & 0xFF) / 255,
            green: Double((rgb >> 8) & 0xFF) / 255,
            blue: Double(rgb & 0xFF) / 255
        )
    }
}
