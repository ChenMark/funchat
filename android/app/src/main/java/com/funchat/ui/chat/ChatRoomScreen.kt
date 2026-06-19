// FunChat Android - 聊天详情页 (CHAT-010)
// 消息气泡 + 输入框 + WebSocket 实时收发
// 基于 DES-003 设计稿 P2

package com.funchat.ui.chat

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Send
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.funchat.data.api.APIClient
import com.funchat.ui.auth.PrimaryColor
import com.funchat.ui.auth.PrimaryDarkColor
import com.funchat.ui.auth.Neutral100
import com.funchat.ui.auth.Neutral50
import com.funchat.ui.auth.Neutral400
import com.funchat.ui.auth.Neutral600
import com.funchat.ui.auth.Neutral800
import com.funchat.ui.auth.Neutral900
import com.funchat.ui.auth.White
import kotlinx.coroutines.launch
import okhttp3.*
import okhttp3.WebSocket
import okio.ByteString
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.*

// MARK: - 数据模型

data class ChatMessage(
    val id: Long,
    val fromUserId: String,
    val toUserId: String,
    val msgType: Int,
    val content: String,
    val isBurn: Boolean,
    val burnDuration: Int,
    val timestamp: Long,
    val isMe: Boolean,
    var sendStatus: SendStatus = SendStatus.SENT  // BUG-003 修复
)

enum class SendStatus {
    SENDING,  // 发送中
    SENT,     // 已发送
    FAILED    // 发送失败
}

// MARK: - 聊天详情页

@Composable
fun ChatRoomScreen(
    friendId: String,
    friendName: String,
    viewModel: ChatViewModel = viewModel()
) {
    val messages by viewModel.messages.collectAsState()
    val isConnected by viewModel.isConnected.collectAsState()
    val inputText by viewModel.inputText.collectAsState()
    val listState = rememberLazyListState()
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        viewModel.connect(friendId)
        viewModel.loadHistory(friendId)
    }

    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(messages.size - 1)
        }
    }

    Column(modifier = Modifier.fillMaxSize().background(White)) {
        // 连接状态
        if (!isConnected) {
            Row(
                modifier = Modifier.fillMaxWidth().background(Neutral50).padding(vertical = 4.dp),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically
            ) {
                CircularProgressIndicator(modifier = Modifier.size(12.dp), strokeWidth = 1.dp)
                Spacer(modifier = Modifier.width(4.dp))
                Text("连接中...", fontSize = 12.sp, color = Neutral400)
            }
        }

        // 消息列表
        LazyColumn(
            modifier = Modifier.weight(1f).fillMaxWidth(),
            state = listState,
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            items(messages) { msg ->
                MessageBubble(message = msg)
            }
        }

        // 输入栏
        ChatInputBar(
            text = inputText,
            onTextChange = { viewModel.updateInput(it) },
            onSend = {
                viewModel.sendMessage(friendId)
            },
            onMore = { /* TODO: 更多菜单 */ }
        )
    }
}

// MARK: - 消息气泡

@Composable
fun MessageBubble(message: ChatMessage, onRetry: ((ChatMessage) -> Unit)? = null) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (message.isMe) Arrangement.End else Arrangement.Start
    ) {
        Box(
            modifier = Modifier
                .clip(
                    RoundedCornerShape(
                        topStart = 12.dp,
                        topEnd = 12.dp,
                        bottomStart = if (message.isMe) 12.dp else 4.dp,
                        bottomEnd = if (message.isMe) 4.dp else 12.dp
                    )
                )
                .background(
                    if (message.isBurn) {
                        Neutral900
                    } else if (message.isMe) {
                        Brush.horizontalGradient(listOf(PrimaryColor, PrimaryDarkColor))
                    } else {
                        Brush.horizontalGradient(listOf(Neutral100, Neutral100))
                    }
                )
                .padding(horizontal = 12.dp, vertical = 10.dp)
                .widthIn(max = 250.dp)
                .then(
                    if (message.sendStatus == SendStatus.SENDING)
                        Modifier.alpha(0.5f) else Modifier
                )
        ) {
            Column {
                Text(
                    message.content,
                    fontSize = 15.sp,
                    color = if (message.isBurn || message.isMe) White else Neutral800
                )
                Spacer(modifier = Modifier.height(4.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    if (message.sendStatus == SendStatus.SENDING) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(10.dp),
                            strokeWidth = 1.dp,
                            color = White
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                    }
                    if (message.sendStatus == SendStatus.FAILED && onRetry != null) {
                        IconButton(
                            onClick = { onRetry(message) },
                            modifier = Modifier.size(16.dp)
                        ) {
                            Text("❗", fontSize = 12.sp)
                        }
                        Spacer(modifier = Modifier.width(4.dp))
                    }
                    Text(
                        formatTimestamp(message.timestamp),
                        fontSize = 11.sp,
                        color = Neutral400
                    )
                }
            }
        }
    }
}

// MARK: - 输入栏

@Composable
fun ChatInputBar(
    text: String,
    onTextChange: (String) -> Unit,
    onSend: () -> Unit,
    onMore: () -> Void
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(Neutral50)
            .padding(horizontal = 16.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        BasicTextField(
            value = text,
            onValueChange = onTextChange,
            textStyle = TextStyle(fontSize = 15.sp, color = Neutral800),
            modifier = Modifier
                .weight(1f)
                .background(White, RoundedCornerShape(12.dp))
                .padding(horizontal = 16.dp, vertical = 10.dp),
            decorationBox = { innerTextField ->
                if (text.isEmpty()) {
                    Text("输入消息...", color = Neutral400, fontSize = 15.sp)
                }
                innerTextField()
            }
        )

        Spacer(modifier = Modifier.width(8.dp))

        if (text.isBlank()) {
            IconButton(onClick = onMore, modifier = Modifier.size(32.dp)) {
                Icon(Icons.Default.Add, contentDescription = "更多", tint = Neutral400)
            }
        } else {
            IconButton(
                onClick = onSend,
                modifier = Modifier
                    .size(32.dp)
                    .clip(RoundedCornerShape(50))
                    .background(PrimaryColor)
            ) {
                Icon(Icons.Default.Send, contentDescription = "发送", tint = White, modifier = Modifier.size(16.dp))
            }
        }
    }
}

// MARK: - ViewModel

class ChatViewModel : androidx.lifecycle.ViewModel() {
    private val _messages = kotlinx.coroutines.flow.MutableStateFlow<List<ChatMessage>>(emptyList())
    val messages = _messages.asStateFlow()

    private val _isConnected = kotlinx.coroutines.flow.MutableStateFlow(false)
    val isConnected = _isConnected.asStateFlow()

    private val _inputText = kotlinx.coroutines.flow.MutableStateFlow("")
    val inputText = _inputText.asStateFlow()

    private var webSocket: WebSocket? = null
    private val client = OkHttpClient()

    fun connect(friendId: String) {
        val token = APIClient.accessToken ?: return
        val userId = APIClient.userId ?: return

        val request = Request.Builder()
            .url("ws://10.0.2.2:8080/ws?token=$token")
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                _isConnected.value = true
                // 心跳
                viewModelScope.launch {
                    while (true) {
                        kotlinx.coroutines.delay(30000)
                        webSocket.send("""{"type":"ping"}""")
                    }
                }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text, userId)
            }

            override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
                handleMessage(bytes.utf8(), userId)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                _isConnected.value = false
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                _isConnected.value = false
            }
        })
    }

    fun disconnect() {
        webSocket?.close(1000, "用户离开")
        _isConnected.value = false
    }

    fun updateInput(text: String) {
        _inputText.value = text
    }

    fun sendMessage(friendId: String) {
        val content = _inputText.value.trim()
        if (content.isEmpty()) return

        val userId = APIClient.userId ?: return

        val msgId = System.currentTimeMillis()

        // 先添加本地消息 (sending 状态)
        val localMsg = ChatMessage(
            id = msgId,
            fromUserId = userId,
            toUserId = friendId,
            msgType = 1,
            content = content,
            isBurn = false,
            burnDuration = 0,
            timestamp = System.currentTimeMillis() / 1000,
            isMe = true,
            sendStatus = SendStatus.SENDING
        )
        _messages.value = _messages.value + localMsg
        _inputText.value = ""

        // WebSocket 发送
        val msg = JSONObject().apply {
            put("type", "chat")
            put("to", friendId)
            put("msg_type", 1)
            put("content", content)
        }
        webSocket?.send(msg.toString())

        // HTTP API 发送（持久化）
        viewModelScope.launch {
            try {
                APIClient.sendFriendRequest(friendId, content)
                // 成功 → 更新状态
                _messages.value = _messages.value.map {
                    if (it.id == msgId) it.copy(sendStatus = SendStatus.SENT) else it
                }
            } catch (_: Exception) {
                // 失败 → 标记 failed
                _messages.value = _messages.value.map {
                    if (it.id == msgId) it.copy(sendStatus = SendStatus.FAILED) else it
                }
            }
        }
    }

    /// BUG-003 修复: 重试发送失败的消息
    fun retryMessage(msg: ChatMessage) {
        _messages.value = _messages.value.filter { it.id != msg.id }
        _inputText.value = msg.content
        sendMessage(msg.toUserId)
    }

    fun loadHistory(friendId: String) {
        val userId = APIClient.userId ?: return
        viewModelScope.launch {
            try {
                // TODO: 调用 /messages/history API
                // 暂时跳过
            } catch (_: Exception) {}
        }
    }

    private fun handleMessage(text: String, currentUserId: String) {
        try {
            val json = JSONObject(text)
            val type = json.optString("type")

            when (type) {
                "chat" -> {
                    val msg = ChatMessage(
                        id = json.optLong("msg_id", System.currentTimeMillis()),
                        fromUserId = json.optString("from"),
                        toUserId = json.optString("to"),
                        msgType = json.optInt("msg_type", 1),
                        content = json.optString("content"),
                        isBurn = false,
                        burnDuration = 0,
                        timestamp = json.optLong("timestamp", System.currentTimeMillis() / 1000),
                        isMe = json.optString("from") == currentUserId
                    )
                    _messages.value = _messages.value + msg
                }
                "pong" -> { /* 心跳回复 */ }
                "ack" -> { /* 消息确认 */ }
            }
        } catch (_: Exception) {}
    }

    override fun onCleared() {
        disconnect()
        super.onCleared()
    }
}

fun formatTimestamp(ts: Long): String {
    val date = Date(ts * 1000)
    val sdf = SimpleDateFormat("HH:mm", Locale.getDefault())
    return sdf.format(date)
}
