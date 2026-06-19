// FunChat iOS - 登录方式选择页 (AUTH-007)
// 微信登录 + Apple ID 登录
// 基于 DES-002 设计稿 P2

import SwiftUI
import AuthenticationServices

struct LoginMethodView: View {
    @State private var agreedToTerms = false
    @State private var showPhoneInput = false
    @State private var showBindPhone = false
    @State private var bindToken = ""
    @State private var showToast = false
    @State private var toastMessage = ""

    var body: some View {
        VStack(spacing: 0) {
            // 页面标题
            VStack(alignment: .leading, spacing: 8) {
                Text("欢迎来到 FunChat")
                    .font(.system(size: 28, weight: .bold))
                    .foregroundColor(Color("neutral-900"))
                Text("选择登录方式")
                    .font(.system(size: 15))
                    .foregroundColor(Color("neutral-600"))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 16)
            .padding(.top, 32)

            // 登录卡片
            VStack(spacing: 16) {
                // 手机号登录
                LoginCard(
                    icon: "📱",
                    title: "手机号登录",
                    subtitle: "使用短信验证码",
                    isEnabled: agreedToTerms
                ) {
                    showPhoneInput = true
                }

                // 微信登录
                LoginCard(
                    icon: "💬",
                    title: "微信登录",
                    subtitle: "使用微信账号",
                    isEnabled: agreedToTerms
                ) {
                    wechatLogin()
                }

                // Apple 登录 (仅 iOS)
                LoginCard(
                    icon: "",
                    title: "Apple 登录",
                    subtitle: "使用 Apple ID",
                    isEnabled: agreedToTerms
                ) {
                    appleLogin()
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 32)

            Spacer()

            // 协议勾选区
            VStack(spacing: 8) {
                HStack(spacing: 6) {
                    Button(action: { agreedToTerms.toggle() }) {
                        Image(systemName: agreedToTerms ? "checkmark.circle.fill" : "circle")
                            .foregroundColor(agreedToTerms ? Color(hex: "FF6B4A") : Color("neutral-200"))
                            .font(.system(size: 16))
                    }
                    Group {
                        Text("我已阅读并同意")
                            .foregroundColor(Color("neutral-600"))
                        Text("《用户协议》").foregroundColor(Color(hex: "007AFF")).underline()
                        Text("和").foregroundColor(Color("neutral-600"))
                        Text("《隐私政策》").foregroundColor(Color(hex: "007AFF")).underline()
                    }
                    .font(.system(size: 11))
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 16)
            .background(Color("neutral-0"))
            .overlay(
                Rectangle().fill(Color("neutral-100")).frame(height: 0.5),
                alignment: .top
            )
        }
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(isPresented: $showPhoneInput) {
            PhoneInputView()
        }
        .navigationDestination(isPresented: $showBindPhone) {
            // 绑定手机号页面（复用 PhoneInputView 流程）
            PhoneInputView()
        }
        .overlay {
            if showToast {
                ToastView(message: toastMessage)
                    .transition(.opacity)
            }
        }
    }

    // MARK: - 微信登录
    private func wechatLogin() {
        // TODO: 调用微信 SDK SendAuthReq
        // WXSendAuthReq { scope: "snsapi_userinfo" }
        // 在 AppDelegate/SceneDelegate 回调中获取 code

        // 模拟流程
        let wechatCode = "wx_mock_code_\(Int.random(in: 1000...9999))"

        Task {
            do {
                let result = try await APIClient.shared.wechatLogin(code: wechatCode)
                await MainActor.run {
                    if result.needBind == true {
                        bindToken = result.bindToken ?? ""
                        showBindPhone = true
                    } else {
                        APIClient.shared.saveAuth(result)
                        // 跳转首页
                    }
                }
            } catch let error as APIError {
                await MainActor.run {
                    toastMessage = error.message
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                        showToast = false
                    }
                }
            } catch {
                await MainActor.run {
                    toastMessage = "网络异常，请检查网络"
                    showToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                        showToast = false
                    }
                }
            }
        }
    }

    // MARK: - Apple ID 登录
    private func appleLogin() {
        let request = ASAuthorizationAppleIDProvider().createRequest()
        request.requestedScopes = [.fullName, .email]

        let controller = ASAuthorizationController(authorizationRequests: [request])
        // controller.delegate = self  // 需在 View 中设置
        controller.performRequests()
    }
}

// MARK: - 登录卡片组件

struct LoginCard: View {
    let icon: String
    let title: String
    let subtitle: String
    let isEnabled: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Text(icon)
                    .font(.system(size: 24))
                    .frame(width: 28, height: 28)
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(Color("neutral-800"))
                    Text(subtitle)
                        .font(.system(size: 13))
                        .foregroundColor(Color("neutral-600"))
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.system(size: 14))
                    .foregroundColor(Color("neutral-400"))
            }
            .padding(.horizontal, 16)
            .frame(height: 72)
            .background(
                RoundedRectangle(cornerRadius: 12)
                    .fill(Color("neutral-0"))
                    .shadow(color: Color.black.opacity(0.06), radius: 8, y: 2)
            )
            .opacity(isEnabled ? 1.0 : 0.4)
        }
        .disabled(!isEnabled)
        .buttonStyle(PlainButtonStyle())
    }
}

// MARK: - Toast 组件

struct ToastView: View {
    let message: String

    var body: some View {
        VStack {
            HStack(spacing: 8) {
                Image(systemName: "exclamationmark.circle.fill")
                    .foregroundColor(.red)
                Text(message)
                    .font(.system(size: 14))
                    .foregroundColor(Color("neutral-800"))
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Color("neutral-100"))
            .cornerRadius(8)
            .padding(.top, 72)
            Spacer()
        }
    }
}

// MARK: - Apple Sign In Delegate (在 ViewModel 中实现)

/*
class AppleSignInDelegate: NSObject, ASAuthorizationControllerDelegate {
    func authorizationController(controller: ASAuthorizationController,
                                  didCompleteWithAuthorization authorization: ASAuthorization) {
        guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
              let identityToken = credential.identityToken else { return }

        let tokenString = String(data: identityToken, encoding: .utf8) ?? ""

        Task {
            do {
                let result = try await APIClient.shared.appleLogin(identityToken: tokenString)
                if result.needBind == true {
                    // 跳转绑定手机号页面
                } else {
                    APIClient.shared.saveAuth(result)
                    // 跳转首页
                }
            } catch {
                // 错误处理
            }
        }
    }

    func authorizationController(controller: ASAuthorizationController, didCompleteWithError error: Error) {
        // 用户取消或授权失败
    }
}
*/
