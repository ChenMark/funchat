// FunChat iOS - API 客户端
// 对接后端 Go + Gin API

import Foundation

// MARK: - API 响应模型

struct APIResponse<T: Codable>: Codable {
    let code: Int
    let message: String
    let data: T?
}

struct AuthResult: Codable {
    let userId: String?
    let nickname: String?
    let phone: String?
    let accessToken: String?
    let refreshToken: String?
    let expiresIn: Int?
    let codeToken: String?
    let isNew: Bool?
    let bindToken: String?
    let needBind: Bool?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case nickname
        case phone
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case expiresIn = "expires_in"
        case codeToken = "code_token"
        case isNew = "is_new"
        case bindToken = "bind_token"
        case needBind = "need_bind"
    }
}

struct SendCodeResult: Codable {
    let message: String
    let code: String?  // 开发环境返回
}

struct SearchResult: Codable {
    let keyword: String
    let users: [SearchUser]
}

struct SearchUser: Codable, Identifiable {
    let userId: String
    let nickname: String
    let avatar: String
    let isFriend: Bool

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case nickname
        case avatar
        case isFriend = "is_friend"
    }
}

struct FriendRequestItem: Codable, Identifiable {
    let id: Int64
    let fromUserId: String
    let toUserId: String
    let message: String
    let status: Int8
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case fromUserId = "from_user_id"
        case toUserId = "to_user_id"
        case message
        case status
        case createdAt = "created_at"
    }
}

struct FriendItem: Codable, Identifiable {
    let userId: String
    let nickname: String
    let avatar: String
    let lastMsg: String
    let lastTime: String
    let unread: Int

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case nickname
        case avatar
        case lastMsg = "last_msg"
        case lastTime = "last_time"
        case unread
    }
}

// MARK: - API 错误

struct APIError: Error, LocalizedError {
    let code: Int
    let message: String

    var errorDescription: String? { message }
}

// MARK: - API 客户端

class APIClient {
    static let shared = APIClient()

    #if DEBUG
    private let baseURL = "http://localhost:8080/api/v1"
    #else
    private let baseURL = "https://api.funchat.com/api/v1"
    #endif

    private let session: URLSession
    private let decoder = JSONDecoder()

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 15
        session = URLSession(configuration: config)
    }

    // MARK: - Token 管理

    var accessToken: String? {
        get { UserDefaults.standard.string(forKey: "access_token") }
        set { UserDefaults.standard.set(newValue, forKey: "access_token") }
    }

    var refreshToken: String? {
        get { UserDefaults.standard.string(forKey: "refresh_token") }
        set { UserDefaults.standard.set(newValue, forKey: "refresh_token") }
    }

    var isLoggedIn: Bool { accessToken != nil }

    // MARK: - 请求方法

    func get<T: Codable>(_ path: String, query: [String: String] = [:]) async throws -> T {
        try await request("GET", path, query: query)
    }

    func post<T: Codable>(_ path: String, body: [String: Any]? = nil) async throws -> T {
        try await request("POST", path, body: body)
    }

    func delete<T: Codable>(_ path: String) async throws -> T {
        try await request("DELETE", path)
    }

    private func request<T: Codable>(_ method: String, _ path: String,
                                       query: [String: String] = [:],
                                       body: [String: Any]? = nil) async throws -> T {

        var urlComponents = URLComponents(string: baseURL + path)!
        if !query.isEmpty {
            urlComponents.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }

        var req = URLRequest(url: urlComponents.url!)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if let token = accessToken {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body = body {
            req.httpBody = try JSONSerialization.data(withJSONObject: body)
        }

        let (data, response) = try await session.data(for: req)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError(code: -1, message: "网络异常")
        }

        let apiResponse = try decoder.decode(APIResponse<T>.self, from: data)

        if apiResponse.code != 0 {
            throw APIError(code: apiResponse.code, message: apiResponse.message)
        }

        guard let result = apiResponse.data else {
            throw APIError(code: -1, message: "响应数据为空")
        }

        return result
    }
}

// MARK: - Auth API

extension APIClient {

    /// 发送验证码
    func sendCode(phone: String) async throws -> SendCodeResult {
        try await post("/auth/send-code", body: ["phone": phone])
    }

    /// 校验验证码
    func verifyCode(phone: String, code: String) async throws -> AuthResult {
        try await post("/auth/verify-code", body: ["phone": phone, "code": code])
    }

    /// 注册
    func register(phone: String, codeToken: String, nickname: String?, avatar: String?) async throws -> AuthResult {
        var body: [String: Any] = ["phone": phone, "code_token": codeToken]
        if let nickname = nickname { body["nickname"] = nickname }
        if let avatar = avatar { body["avatar"] = avatar }
        return try await post("/auth/register", body: body)
    }

    /// 登录
    func login(phone: String, codeToken: String) async throws -> AuthResult {
        try await post("/auth/login", body: ["phone": phone, "code_token": codeToken])
    }

    /// 微信登录
    func wechatLogin(code: String) async throws -> AuthResult {
        try await post("/auth/wechat/login", body: ["code": code])
    }

    /// 微信绑定手机号
    func wechatBind(bindToken: String, phone: String, codeToken: String, nickname: String?) async throws -> AuthResult {
        var body: [String: Any] = ["bind_token": bindToken, "phone": phone, "code_token": codeToken]
        if let nickname = nickname { body["nickname"] = nickname }
        return try await post("/auth/wechat/bind", body: body)
    }

    /// Apple 登录
    func appleLogin(identityToken: String) async throws -> AuthResult {
        try await post("/auth/apple/login", body: ["identity_token": identityToken])
    }

    /// 刷新 Token
    /// 注意：与上面的 `refreshToken` 属性同名会让调用点产生歧义，故方法名加 Auth 前缀
    func refreshAuthToken() async throws -> AuthResult {
        try await post("/auth/refresh", body: ["refresh_token": refreshToken ?? ""])
    }

    /// 登出
    func logout() async {
        let _: APIResponse<EmptyData>? = try? await post("/auth/logout", body: nil)
        accessToken = nil
        refreshToken = nil
    }

    /// 保存登录信息
    func saveAuth(_ result: AuthResult) {
        accessToken = result.accessToken
        refreshToken = result.refreshToken
        if let userId = result.userId {
            UserDefaults.standard.set(userId, forKey: "user_id")
        }
        if let nickname = result.nickname {
            UserDefaults.standard.set(nickname, forKey: "nickname")
        }
    }
}

struct EmptyData: Codable {}

// MARK: - Friend API

extension APIClient {

    /// 搜索好友
    func searchFriends(keyword: String) async throws -> SearchResult {
        try await get("/friends/search", query: ["keyword": keyword])
    }

    /// 发送好友申请
    func sendFriendRequest(targetUserId: String, message: String?) async throws -> [String: String] {
        var body: [String: Any] = ["target_user_id": targetUserId]
        if let message = message { body["message"] = message }
        return try await post("/friends/request", body: body)
    }

    /// 获取好友申请列表
    func getFriendRequests() async throws -> [FriendRequestItem] {
        struct Wrapper: Codable { let requests: [FriendRequestItem] }
        let wrapper: Wrapper = try await get("/friends/requests")
        return wrapper.requests
    }

    /// 同意好友申请
    func acceptFriendRequest(requestId: Int64) async throws -> [String: String] {
        try await post("/friends/accept", body: ["request_id": requestId])
    }

    /// 拒绝好友申请
    func rejectFriendRequest(requestId: Int64) async throws -> [String: String] {
        try await post("/friends/reject", body: ["request_id": requestId])
    }

    /// 好友列表
    func getFriends() async throws -> [FriendItem] {
        struct Wrapper: Codable { let friends: [FriendItem] }
        let wrapper: Wrapper = try await get("/friends")
        return wrapper.friends
    }

    /// 删除好友
    func deleteFriend(friendId: String) async throws -> [String: String] {
        try await delete("/friends/\(friendId)")
    }
}
