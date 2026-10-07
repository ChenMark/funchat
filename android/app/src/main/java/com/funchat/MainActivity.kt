package com.funchat

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.compose.material.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import com.funchat.data.api.FriendItem
import com.funchat.ui.auth.CodeInputScreen
import com.funchat.ui.auth.LoginMethodScreen
import com.funchat.ui.auth.PhoneInputScreen
import com.funchat.ui.chat.ChatRoomScreen
import com.funchat.ui.friends.AddFriendScreen
import com.funchat.ui.friends.ChatListScreen

// FunChat Android - 工程化补齐的最小可启动入口（事项 rj8tx9）
//
// 说明：本文件只负责「能启动 + 页面能串起来」，不承载业务逻辑。
// 现有 Screen 之间靠回调导航（未引入 navigation-compose），因此这里用一个状态机串联。
// TODO: 页面变多后迁移到 navigation-compose。

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // TODO(rz7mgu): 防截图 —— 阅后即焚要求屏蔽系统截图/最近任务缩略图
        // window.setFlags(
        //     WindowManager.LayoutParams.FLAG_SECURE,
        //     WindowManager.LayoutParams.FLAG_SECURE
        // )
        setContent {
            MaterialTheme {
                FunChatNav()
            }
        }
    }
}

private enum class Screen { LoginMethod, PhoneInput, CodeInput, Home, AddFriend, ChatRoom }

@Composable
private fun FunChatNav() {
    var screen by remember { mutableStateOf(Screen.LoginMethod) }
    var phone by remember { mutableStateOf("") }
    var initialCode by remember { mutableStateOf("") }
    var chatFriendId by remember { mutableStateOf("") }
    var chatFriendName by remember { mutableStateOf("") }

    // 登录方式页是根页面，不再往回退
    BackHandler(enabled = screen != Screen.LoginMethod) {
        screen = when (screen) {
            Screen.PhoneInput, Screen.CodeInput -> Screen.LoginMethod
            Screen.AddFriend, Screen.ChatRoom -> Screen.Home
            else -> Screen.LoginMethod
        }
    }

    when (screen) {
        Screen.LoginMethod -> LoginMethodScreen(
            onPhoneLogin = { screen = Screen.PhoneInput },
            // TODO(V1.1): 微信 SDK 接入后实现；当前后端生产环境返回 501/50100
            onWechatLogin = { /* no-op */ }
        )

        Screen.PhoneInput -> PhoneInputScreen(
            onNavigateToCode = { p, code ->
                phone = p
                initialCode = code
                screen = Screen.CodeInput
            }
        )

        Screen.CodeInput -> CodeInputScreen(
            phone = phone,
            initialCode = initialCode,
            onLoginSuccess = { _ -> screen = Screen.Home }
        )

        Screen.Home -> ChatListScreen(
            onChatClick = { friend: FriendItem ->
                chatFriendId = friend.userId
                chatFriendName = friend.nickname
                screen = Screen.ChatRoom
            },
            onAddFriend = { screen = Screen.AddFriend }
        )

        Screen.AddFriend -> AddFriendScreen()

        Screen.ChatRoom -> ChatRoomScreen(
            friendId = chatFriendId,
            friendName = chatFriendName
        )
    }
}
