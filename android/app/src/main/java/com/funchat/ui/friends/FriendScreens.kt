// FunChat Android - 好友列表 + 添加好友 UI (FRD-005 + FRD-006)
// 基于 DES-003 设计稿

package com.funchat.ui.friends

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Search
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.funchat.data.api.APIClient
import com.funchat.data.api.FriendItem
import com.funchat.data.api.FriendRequestItem
import com.funchat.data.api.SearchUser
import kotlinx.coroutines.launch

// MARK: - 好友列表页

@Composable
fun ChatListScreen(
    onChatClick: (FriendItem) -> Unit,
    onAddFriend: () -> Unit,
    viewModel: FriendViewModel = viewModel()
) {
    val friends by viewModel.friends.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()

    LaunchedEffect(Unit) {
        viewModel.loadFriends()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("聊天", fontSize = 20.sp, fontWeight = FontWeight.Bold) },
                backgroundColor = Color.White,
                elevation = 0.dp,
                actions = {
                    IconButton(onClick = onAddFriend) {
                        Icon(Icons.Default.Add, contentDescription = "添加好友")
                    }
                }
            )
        }
    ) { padding ->
        if (friends.isEmpty() && !isLoading) {
            // 空状态
            Column(
                modifier = Modifier.fillMaxSize().padding(padding),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center
            ) {
                Box(
                    modifier = Modifier.size(200.dp).clip(CircleShape)
                        .background(Color(0xFFFFF0ED)),
                    contentAlignment = Alignment.Center
                ) {
                    Text("💬", fontSize = 64.sp)
                }
                Spacer(modifier = Modifier.height(16.dp))
                Text("还没有聊天", fontSize = 18.sp, fontWeight = FontWeight.SemiBold, color = Color(0xFF3A3A3C))
                Text("去添加好友吧", fontSize = 15.sp, color = Color(0xFF636366))
                Spacer(modifier = Modifier.height(16.dp))
                Button(
                    onClick = onAddFriend,
                    colors = ButtonDefaults.buttonColors(backgroundColor = Color(0xFFFFD9D1)),
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.width(140.dp).height(44.dp)
                ) {
                    Text("添加好友", color = Color(0xFFFF6B4A), fontWeight = FontWeight.SemiBold)
                }
            }
        } else {
            // 好友列表
            LazyColumn(modifier = Modifier.padding(padding)) {
                items(friends) { friend ->
                    FriendRow(friend, onClick = { onChatClick(friend) })
                    Divider(color = Color(0xFFE8E8ED), thickness = 0.5.dp,
                        modifier = Modifier.padding(start = 64.dp))
                }
            }
        }
    }
}

@Composable
private fun FriendRow(friend: FriendItem, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(72.dp)
            .padding(horizontal = 16.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // 头像
        Box(
            modifier = Modifier.size(36.dp).clip(CircleShape)
                .background(Color(0xFFFFD9D1)),
            contentAlignment = Alignment.Center
        ) {
            Text(
                friend.nickname.take(1),
                color = Color(0xFFFF6B4A),
                fontWeight = FontWeight.Medium
            )
        }
        Spacer(modifier = Modifier.width(12.dp))

        // 昵称 + 最后消息
        Column(modifier = Modifier.weight(1f)) {
            Text(friend.nickname, fontSize = 16.sp, fontWeight = FontWeight.SemiBold, color = Color(0xFF3A3A3C))
            Spacer(modifier = Modifier.height(4.dp))
            Text(friend.lastMsg, fontSize = 13.sp, color = Color(0xFF636366), maxLines = 1)
        }

        // 时间 + 未读
        Column(horizontalAlignment = Alignment.End) {
            Text(friend.lastTime, fontSize = 11.sp, color = Color(0xFF8E8E93))
            if (friend.unread > 0) {
                Spacer(modifier = Modifier.height(4.dp))
                Box(
                    modifier = Modifier.size(18.dp).clip(CircleShape).background(Color.Red),
                    contentAlignment = Alignment.Center
                ) {
                    Text(friend.unread.toString(), color = Color.White, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

// MARK: - 添加好友页

@Composable
fun AddFriendScreen(
    viewModel: FriendViewModel = viewModel()
) {
    var searchText by remember { mutableStateOf("") }
    var searchResults by remember { mutableStateOf<List<SearchUser>>(emptyList()) }
    val requests by viewModel.requests.collectAsState()
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(Unit) {
        viewModel.loadRequests()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("添加好友") },
                backgroundColor = Color.White,
                elevation = 0.dp
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            // 搜索栏
            Row(
                modifier = Modifier.padding(16.dp).fillMaxWidth().height(40.dp)
                    .background(Color(0xFFF5F5F7), RoundedCornerShape(12.dp))
                    .padding(horizontal = 12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(Icons.Default.Search, contentDescription = null, tint = Color(0xFF8E8E93))
                Spacer(modifier = Modifier.width(8.dp))
                TextField(
                    value = searchText,
                    onValueChange = {
                        searchText = it
                        if (it.length >= 2) {
                            scope.launch {
                                try {
                                    searchResults = APIClient.searchFriends(it).users
                                } catch (e: Exception) {
                                    searchResults = emptyList()
                                }
                            }
                        } else {
                            searchResults = emptyList()
                        }
                    },
                    placeholder = { Text("搜索手机号或用户名", color = Color(0xFF8E8E93), fontSize = 15.sp) },
                    colors = TextFieldDefaults.textFieldColors(
                        backgroundColor = Color.Transparent,
                        focusedIndicatorColor = Color.Transparent,
                        unfocusedIndicatorColor = Color.Transparent
                    ),
                    modifier = Modifier.weight(1f)
                )
            }

            // 搜索结果
            searchResults.forEach { user ->
                Row(
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Box(
                        modifier = Modifier.size(36.dp).clip(CircleShape).background(Color(0xFFFFD9D1)),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(user.nickname.take(1), color = Color(0xFFFF6B4A))
                    }
                    Spacer(modifier = Modifier.width(12.dp))
                    Column(modifier = Modifier.weight(1f)) {
                        Text(user.nickname, fontWeight = FontWeight.Medium, color = Color(0xFF3A3A3C))
                        Text("ID: ${user.userId}", fontSize = 11.sp, color = Color(0xFF8E8E93))
                    }
                    if (user.isFriend) {
                        Text("已添加", color = Color(0xFF8E8E93), fontSize = 13.sp)
                    } else {
                        TextButton(onClick = {
                            scope.launch {
                                try {
                                    APIClient.sendFriendRequest(user.userId)
                                    snackbarHostState.showSnackbar("申请已发送")
                                } catch (e: Exception) {
                                    snackbarHostState.showSnackbar(e.message ?: "操作失败")
                                }
                            }
                        }) {
                            Text("添加", color = Color(0xFFFF6B4A), fontWeight = FontWeight.SemiBold)
                        }
                    }
                }
            }

            Divider(modifier = Modifier.padding(vertical = 16.dp))

            // 好友请求
            if (requests.isNotEmpty()) {
                Text("好友请求", fontSize = 11.sp, color = Color(0xFF8E8E93),
                    modifier = Modifier.padding(horizontal = 16.dp))
                requests.forEach { req ->
                    Row(
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Box(
                            modifier = Modifier.size(36.dp).clip(CircleShape).background(Color(0xFFFFD9D1)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(Icons.Default.Person, contentDescription = null, tint = Color(0xFFFF6B4A))
                        }
                        Spacer(modifier = Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(req.fromUserId, fontWeight = FontWeight.Medium)
                            if (req.message.isNotEmpty()) {
                                Text(req.message, fontSize = 13.sp, color = Color(0xFF636366))
                            }
                        }
                        // 接受
                        TextButton(onClick = {
                            scope.launch {
                                try {
                                    APIClient.acceptFriendRequest(req.id)
                                    viewModel.loadRequests()
                                    snackbarHostState.showSnackbar("已添加为好友")
                                } catch (e: Exception) {
                                    snackbarHostState.showSnackbar("操作失败")
                                }
                            }
                        }) { Text("✓", color = Color(0xFF34C759), fontSize = 20.sp, fontWeight = FontWeight.Bold) }

                        // 拒绝
                        TextButton(onClick = {
                            scope.launch {
                                try {
                                    APIClient.rejectFriendRequest(req.id)
                                    viewModel.loadRequests()
                                } catch (e: Exception) {
                                    snackbarHostState.showSnackbar("操作失败")
                                }
                            }
                        }) { Text("✕", color = Color(0xFFFF3B30), fontSize = 20.sp, fontWeight = FontWeight.Bold) }
                    }
                }
            }
        }
    }
}

// MARK: - ViewModel

class FriendViewModel : androidx.lifecycle.ViewModel() {
    private val _friends = kotlinx.coroutines.flow.MutableStateFlow<List<FriendItem>>(emptyList())
    val friends = _friends.asStateFlow()

    private val _requests = kotlinx.coroutines.flow.MutableStateFlow<List<FriendRequestItem>>(emptyList())
    val requests = _requests.asStateFlow()

    private val _isLoading = kotlinx.coroutines.flow.MutableStateFlow(false)
    val isLoading = _isLoading.asStateFlow()

    fun loadFriends() {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                _friends.value = APIClient.getFriends()
            } catch (e: Exception) {
                _friends.value = emptyList()
            }
            _isLoading.value = false
        }
    }

    fun loadRequests() {
        viewModelScope.launch {
            try {
                _requests.value = APIClient.getFriendRequests()
            } catch (e: Exception) {
                _requests.value = emptyList()
            }
        }
    }

    fun deleteFriend(friendId: String) {
        viewModelScope.launch {
            try {
                APIClient.deleteFriend(friendId)
                _friends.value = _friends.value.filter { it.userId != friendId }
            } catch (e: Exception) {
                // 错误处理
            }
        }
    }
}
