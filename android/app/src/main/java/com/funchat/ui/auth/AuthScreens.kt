// FunChat Android - 登录注册 UI (AUTH-006 + AUTH-008)
// 手机号输入 + 验证码 + 信息完善 + 微信登录

package com.funchat.ui.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.launch

// MARK: - 颜色定义

val PrimaryColor = Color(0xFFFF6B4A)
val PrimaryDarkColor = Color(0xFFFF3B30)
val Neutral900 = Color(0xFF1C1C1E)
val Neutral800 = Color(0xFF3A3A3C)
val Neutral600 = Color(0xFF636366)
val Neutral400 = Color(0xFF8E8E93)
val Neutral200 = Color(0xFFD1D1D6)
val Neutral100 = Color(0xFFE8E8ED)
val Neutral50 = Color(0xFFF5F5F7)
val White = Color(0xFFFFFFFF)
val InfoColor = Color(0xFF007AFF)
val ErrorColor = Color(0xFFFF3B30)

// MARK: - 登录方式选择页

@Composable
fun LoginMethodScreen(
    onPhoneLogin: () -> Unit,
    onWechatLogin: () -> Unit,
    viewModel: AuthViewModel = viewModel()
) {
    var agreedToTerms by remember { mutableStateOf(false) }

    Column(modifier = Modifier.fillMaxSize().background(White)) {
        // 标题
        Column(modifier = Modifier.padding(horizontal = 16.dp, vertical = 32.dp)) {
            Text("欢迎来到 FunChat", fontSize = 28.sp, fontWeight = FontWeight.Bold, color = Neutral900)
            Spacer(modifier = Modifier.height(8.dp))
            Text("选择登录方式", fontSize = 15.sp, color = Neutral600)
        }

        // 登录卡片
        Column(modifier = Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            LoginCard("📱", "手机号登录", "使用短信验证码", agreedToTerms, onPhoneLogin)
            LoginCard("💬", "微信登录", "使用微信账号", agreedToTerms) {
                viewModel.wechatLogin()
            }
        }

        Spacer(modifier = Modifier.weight(1f))

        // 协议勾选
        Row(
            modifier = Modifier.fillMaxWidth().padding(16.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Checkbox(
                checked = agreedToTerms,
                onCheckedChange = { agreedToTerms = it },
                colors = CheckboxDefaults.colors(checkedColor = PrimaryColor)
            )
            Text("我已阅读并同意", fontSize = 11.sp, color = Neutral600)
            Text("《用户协议》", fontSize = 11.sp, color = InfoColor)
            Text("和", fontSize = 11.sp, color = Neutral600)
            Text("《隐私政策》", fontSize = 11.sp, color = InfoColor)
        }
    }
}

@Composable
private fun LoginCard(icon: String, title: String, subtitle: String, enabled: Boolean, onClick: () -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth().height(72.dp),
        elevation = 4.dp,
        shape = RoundedCornerShape(12.dp),
        backgroundColor = if (enabled) White else White.copy(alpha = 0.4f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(icon, fontSize = 24.sp)
            Spacer(modifier = Modifier.width(12.dp))
            Column {
                Text(title, fontSize = 16.sp, fontWeight = FontWeight.SemiBold, color = Neutral800)
                Text(subtitle, fontSize = 13.sp, color = Neutral600)
            }
            Spacer(modifier = Modifier.weight(1f))
            Text("›", fontSize = 20.sp, color = Neutral400)
        }
    }
}

// MARK: - 手机号输入页

@Composable
fun PhoneInputScreen(
    onNavigateToCode: (String, String) -> Unit,
    viewModel: AuthViewModel = viewModel()
) {
    var phone by remember { mutableStateOf("") }
    var agreedToTerms by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val isLoading by viewModel.isLoading.collectAsState()

    Column(modifier = Modifier.fillMaxSize().background(White)) {
        // 标题
        Column(modifier = Modifier.padding(horizontal = 16.dp, vertical = 32.dp)) {
            Text("输入手机号", fontSize = 28.sp, fontWeight = FontWeight.Bold, color = Neutral900)
            Spacer(modifier = Modifier.height(8.dp))
            Text("使用短信验证码登录", fontSize = 15.sp, color = Neutral600)
        }

        // 手机号输入
        Box(
            modifier = Modifier
                .padding(horizontal = 16.dp)
                .fillMaxWidth()
                .height(56.dp)
                .background(Neutral50, RoundedCornerShape(8.dp))
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 16.dp).fillMaxHeight(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text("🇨🇳 +86", fontSize = 15.sp, fontWeight = FontWeight.Medium, color = Neutral800)
                Spacer(modifier = Modifier.width(12.dp))
                Box(modifier = Modifier.width(1.dp).height(20.dp).background(Neutral200))
                Spacer(modifier = Modifier.width(12.dp))
                TextField(
                    value = phone,
                    onValueChange = {
                        val digits = it.filter { c -> c.isDigit() }.take(11)
                        phone = digits
                        errorMessage = null
                    },
                    placeholder = { Text("请输入手机号", color = Neutral400) },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    colors = TextFieldDefaults.textFieldColors(
                        backgroundColor = Color.Transparent,
                        focusedIndicatorColor = Color.Transparent,
                        unfocusedIndicatorColor = Color.Transparent
                    ),
                    modifier = Modifier.weight(1f)
                )
            }
        }

        // 错误提示
        errorMessage?.let {
            Text(it, fontSize = 13.sp, color = ErrorColor,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp))
        }

        Spacer(modifier = Modifier.weight(1f))

        // 获取验证码按钮
        val canSend = phone.length == 11 && agreedToTerms && !isLoading
        Button(
            onClick = {
                scope.launch {
                    try {
                        val result = viewModel.sendCode(phone)
                        val code = result.code ?: ""
                        onNavigateToCode(phone, code)
                    } catch (e: Exception) {
                        errorMessage = e.message ?: "网络异常"
                    }
                }
            },
            enabled = canSend,
            modifier = Modifier.padding(horizontal = 16.dp).fillMaxWidth().height(48.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(
                backgroundColor = if (canSend) PrimaryColor else Neutral100,
                contentColor = if (canSend) White else Neutral400
            )
        ) {
            if (isLoading) {
                CircularProgressIndicator(modifier = Modifier.size(20.dp), color = White)
            } else {
                Text("获取验证码", fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
            }
        }

        // 协议
        Row(
            modifier = Modifier.padding(vertical = 16.dp).fillMaxWidth(),
            horizontalArrangement = Arrangement.Center
        ) {
            Checkbox(checked = agreedToTerms, onCheckedChange = { agreedToTerms = it },
                colors = CheckboxDefaults.colors(checkedColor = PrimaryColor))
            Text("我已阅读并同意", fontSize = 11.sp, color = Neutral600)
            Text("《用户协议》", fontSize = 11.sp, color = InfoColor)
            Text("和", fontSize = 11.sp, color = Neutral600)
            Text("《隐私政策》", fontSize = 11.sp, color = InfoColor)
        }
    }
}

// MARK: - 验证码输入页

@Composable
fun CodeInputScreen(
    phone: String,
    initialCode: String,
    onLoginSuccess: (Boolean) -> Unit,
    viewModel: AuthViewModel = viewModel()
) {
    var code by remember { mutableStateOf("") }
    var countdown by remember { mutableStateOf(60) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val isLoading by viewModel.isLoading.collectAsState()

    // 倒计时
    LaunchedEffect(countdown) {
        if (countdown > 0) {
            kotlinx.coroutines.delay(1000)
            countdown--
        }
    }

    Column(modifier = Modifier.fillMaxSize().background(White)) {
        // 标题
        Column(modifier = Modifier.padding(horizontal = 16.dp, vertical = 32.dp)) {
            Text("输入验证码", fontSize = 28.sp, fontWeight = FontWeight.Bold, color = Neutral900)
            Spacer(modifier = Modifier.height(8.dp))
            val masked = if (phone.length == 11) "${phone.take(3)}****${phone.takeLast(4)}" else phone
            Text("已发送至 +86 $masked", fontSize = 15.sp, color = Neutral600)
        }

        // 6 格验证码
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 32.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            repeat(6) { index ->
                val char = if (index < code.length) code[index].toString() else ""
                val isActive = index == code.length
                Box(
                    modifier = Modifier
                        .size(48.dp)
                        .background(
                            if (errorMessage != null) Color(0xFFFFF0F0) else Neutral50,
                            RoundedCornerShape(8.dp)
                        ),
                    contentAlignment = Alignment.Center
                ) {
                    if (char.isNotEmpty()) {
                        Text(char, fontSize = 22.sp, fontWeight = FontWeight.SemiBold, color = Neutral800)
                    }
                }
            }
        }

        // 隐藏输入框
        TextField(
            value = code,
            onValueChange = {
                val digits = it.filter { c -> c.isDigit() }.take(6)
                code = digits
                errorMessage = null
                if (code.length == 6) {
                    scope.launch {
                        try {
                            val isNew = viewModel.verifyAndLogin(phone, code, initialCode)
                            onLoginSuccess(isNew)
                        } catch (e: Exception) {
                            errorMessage = e.message ?: "验证码错误"
                            code = ""
                        }
                    }
                }
            },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.size(0.dp),
            colors = TextFieldDefaults.textFieldColors(
                backgroundColor = Color.Transparent,
                focusedIndicatorColor = Color.Transparent,
                unfocusedIndicatorColor = Color.Transparent
            )
        )

        // 倒计时
        Text(
            if (countdown > 0) "${countdown}s 后重新发送" else "重新发送",
            fontSize = 13.sp,
            color = if (countdown > 0) Neutral600 else InfoColor,
            modifier = Modifier.padding(top = 16.dp)
        )

        Spacer(modifier = Modifier.weight(1f))
    }
}

// MARK: - ViewModel

class AuthViewModel : androidx.lifecycle.ViewModel() {
    private val _isLoading = kotlinx.coroutines.flow.MutableStateFlow(false)
    val isLoading = _isLoading.asStateFlow()

    suspend fun sendCode(phone: String): SendCodeResult {
        _isLoading.value = true
        try {
            return com.funchat.data.api.APIClient.sendCode(phone)
        } finally {
            _isLoading.value = false
        }
    }

    suspend fun verifyAndLogin(phone: String, code: String, initialCode: String): Boolean {
        _isLoading.value = true
        try {
            val verifyResult = com.funchat.data.api.APIClient.verifyCode(phone, code)
            val codeToken = verifyResult.codeToken ?: initialCode
            val isNew = verifyResult.isNew ?: false

            val authResult = if (isNew) {
                com.funchat.data.api.APIClient.register(phone, codeToken)
            } else {
                com.funchat.data.api.APIClient.login(phone, codeToken)
            }

            com.funchat.data.api.APIClient.saveAuth(authResult)
            return isNew
        } finally {
            _isLoading.value = false
        }
    }

    fun wechatLogin() {
        // TODO: 调用微信 SDK SendAuth.Req
        viewModelScope.launch {
            try {
                val code = "wx_mock_code_${(1000..9999).random()}"
                val result = com.funchat.data.api.APIClient.wechatLogin(code)
                if (result.needBind == true) {
                    // 跳转绑定手机号页
                } else {
                    com.funchat.data.api.APIClient.saveAuth(result)
                    // 跳转首页
                }
            } catch (e: Exception) {
                // Toast
            }
        }
    }
}
