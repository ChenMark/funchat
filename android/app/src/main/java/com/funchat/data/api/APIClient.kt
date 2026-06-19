// FunChat Android - API 客户端
// 对接后端 Go + Gin API

package com.funchat.data.api

import com.google.gson.Gson
import com.google.gson.annotations.SerializedName
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.TimeUnit

// MARK: - 响应模型

data class APIResponse<T>(
    val code: Int,
    val message: String,
    val data: T?
)

data class AuthResult(
    @SerializedName("user_id") val userId: String? = null,
    val nickname: String? = null,
    val phone: String? = null,
    @SerializedName("access_token") val accessToken: String? = null,
    @SerializedName("refresh_token") val refreshToken: String? = null,
    @SerializedName("expires_in") val expiresIn: Int? = null,
    @SerializedName("code_token") val codeToken: String? = null,
    @SerializedName("is_new") val isNew: Boolean? = null,
    @SerializedName("bind_token") val bindToken: String? = null,
    @SerializedName("need_bind") val needBind: Boolean? = null
)

data class SendCodeResult(
    val message: String,
    val code: String? = null
)

data class SearchResult(
    val keyword: String,
    val users: List<SearchUser>
)

data class SearchUser(
    @SerializedName("user_id") val userId: String,
    val nickname: String,
    val avatar: String,
    @SerializedName("is_friend") val isFriend: Boolean
)

data class FriendRequestItem(
    val id: Long,
    @SerializedName("from_user_id") val fromUserId: String,
    @SerializedName("to_user_id") val toUserId: String,
    val message: String,
    val status: Int,
    @SerializedName("created_at") val createdAt: String
)

data class FriendItem(
    @SerializedName("user_id") val userId: String,
    val nickname: String,
    val avatar: String,
    @SerializedName("last_msg") val lastMsg: String,
    @SerializedName("last_time") val lastTime: String,
    val unread: Int
)

data class FriendsWrapper(val friends: List<FriendItem>)
data class RequestsWrapper(val requests: List<FriendRequestItem>)

// MARK: - API 异常

class APIException(val code: Int, override val message: String) : Exception(message)

// MARK: - API 客户端

object APIClient {

    private const val BASE_URL = "http://10.0.2.2:8080/api/v1"  // Android 模拟器访问宿主机

    private val client = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(15, TimeUnit.SECONDS)
        .build()

    private val gson = Gson()

    // Token 管理 (简化版，生产环境用 EncryptedSharedPreferences)
    var accessToken: String?
        get() = tokenHolder
        set(value) { tokenHolder = value }

    private var tokenHolder: String? = null
    var refreshToken: String? = null
    var userId: String? = null

    val isLoggedIn: Boolean get() = accessToken != null

    // MARK: - 请求方法

    private suspend fun <T> request(
        method: String,
        path: String,
        query: Map<String, String> = emptyMap(),
        body: Map<String, Any>? = null,
        responseType: Class<T>
    ): T = withContext(Dispatchers.IO) {

        val urlBuilder = StringBuilder(BASE_URL).append(path)
        if (query.isNotEmpty()) {
            urlBuilder.append("?")
            query.entries.joinTo(urlBuilder, "&") { "${it.key}=${it.value}" }
        }

        val requestBuilder = Request.Builder()
            .url(urlBuilder.toString())
            .header("Content-Type", "application/json")

        accessToken?.let { requestBuilder.header("Authorization", "Bearer $it") }

        when (method) {
            "GET" -> requestBuilder.get()
            "DELETE" -> requestBuilder.delete()
            "POST" -> {
                val json = gson.toJson(body ?: emptyMap<String, Any>())
                requestBuilder.post(json.toRequestBody("application/json".toMediaType()))
            }
        }

        val response = client.newCall(requestBuilder.build()).execute()
        val responseBody = response.body?.string() ?: ""

        val apiResponse = gson.fromJson(responseBody, APIResponse::class.java)
            ?: throw APIException(-1, "解析失败")

        if (apiResponse.code != 0) {
            throw APIException(apiResponse.code, apiResponse.message)
        }

        val dataJson = gson.toJson(apiResponse.data)
        gson.fromJson(dataJson, responseType)
            ?: throw APIException(-1, "数据为空")
    }

    // MARK: - Auth API

    suspend fun sendCode(phone: String): SendCodeResult =
        request("POST", "/auth/send-code",
            body = mapOf("phone" to phone),
            responseType = SendCodeResult::class.java)

    suspend fun verifyCode(phone: String, code: String): AuthResult =
        request("POST", "/auth/verify-code",
            body = mapOf("phone" to phone, "code" to code),
            responseType = AuthResult::class.java)

    suspend fun register(phone: String, codeToken: String, nickname: String? = null): AuthResult {
        val body = mutableMapOf("phone" to phone, "code_token" to codeToken)
        nickname?.let { body["nickname"] = it }
        return request("POST", "/auth/register", body = body, responseType = AuthResult::class.java)
    }

    suspend fun login(phone: String, codeToken: String): AuthResult =
        request("POST", "/auth/login",
            body = mapOf("phone" to phone, "code_token" to codeToken),
            responseType = AuthResult::class.java)

    suspend fun wechatLogin(code: String): AuthResult =
        request("POST", "/auth/wechat/login",
            body = mapOf("code" to code),
            responseType = AuthResult::class.java)

    suspend fun wechatBind(bindToken: String, phone: String, codeToken: String): AuthResult =
        request("POST", "/auth/wechat/bind",
            body = mapOf("bind_token" to bindToken, "phone" to phone, "code_token" to codeToken),
            responseType = AuthResult::class.java)

    suspend fun appleLogin(identityToken: String): AuthResult =
        request("POST", "/auth/apple/login",
            body = mapOf("identity_token" to identityToken),
            responseType = AuthResult::class.java)

    suspend fun refreshToken(): AuthResult =
        request("POST", "/auth/refresh",
            body = mapOf("refresh_token" to (refreshToken ?: "")),
            responseType = AuthResult::class.java)

    fun saveAuth(result: AuthResult) {
        accessToken = result.accessToken
        refreshToken = result.refreshToken
        userId = result.userId
    }

    fun logout() {
        accessToken = null
        refreshToken = null
        userId = null
    }

    // MARK: - Friend API

    suspend fun searchFriends(keyword: String): SearchResult =
        request("GET", "/friends/search",
            query = mapOf("keyword" to keyword),
            responseType = SearchResult::class.java)

    suspend fun sendFriendRequest(targetUserId: String, message: String? = null): Map<String, String> {
        val body = mutableMapOf("target_user_id" to targetUserId)
        message?.let { body["message"] = it }
        return request("POST", "/friends/request", body = body, responseType = Map<String, String>::class.java)
    }

    suspend fun getFriendRequests(): List<FriendRequestItem> =
        request("GET", "/friends/requests", responseType = RequestsWrapper::class.java).requests

    suspend fun acceptFriendRequest(requestId: Long): Map<String, String> =
        request("POST", "/friends/accept",
            body = mapOf("request_id" to requestId),
            responseType = Map<String, String>::class.java)

    suspend fun rejectFriendRequest(requestId: Long): Map<String, String> =
        request("POST", "/friends/reject",
            body = mapOf("request_id" to requestId),
            responseType = Map<String, String>::class.java)

    suspend fun getFriends(): List<FriendItem> =
        request("GET", "/friends", responseType = FriendsWrapper::class.java).friends

    suspend fun deleteFriend(friendId: String): Map<String, String> =
        request("DELETE", "/friends/$friendId", responseType = Map<String, String>::class.java)
}
