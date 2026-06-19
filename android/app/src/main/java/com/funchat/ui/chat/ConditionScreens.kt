// FunChat Android - 解锁条件 UI (COND-005/007 + LOC-002 + QUIZ-001)
// 基于 DES-003 设计稿

package com.funchat.ui.chat

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
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

// MARK: - 条件设置入口 (COND-005)

@Composable
fun ConditionSettingsScreen(
    friendId: String,
    onBack: () -> Unit
) {
    var locationEnabled by remember { mutableStateOf(false) }
    var stepsEnabled by remember { mutableStateOf(false) }
    var quizEnabled by remember { mutableStateOf(false) }
    var showLocationPicker by remember { mutableStateOf(false) }
    var showQuizSetup by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("设置解锁条件") },
                backgroundColor = White,
                elevation = 0.dp,
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, "返回")
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp)
        ) {
            Text("选择对方需要满足的条件才能查看你的消息",
                fontSize = 15.sp, color = Neutral600,
                modifier = Modifier.padding(top = 24.dp))

            Spacer(modifier = Modifier.height(24.dp))

            // 定位
            ConditionCard("location.fill", "定位解锁", "需要在一定范围内", Color(0xFF20C0B0),
                locationEnabled, { locationEnabled = it }) { showLocationPicker = true }

            Spacer(modifier = Modifier.height(16.dp))

            // 步数
            ConditionCard("figure.walk", "步数解锁", "需要达到目标步数", Color(0xFFFFB020),
                stepsEnabled, { stepsEnabled = it }) { /* picker */ }

            Spacer(modifier = Modifier.height(16.dp))

            // 答题
            ConditionCard("brain.head.profile", "答题解锁", "需要答对你的问题", Color(0xFF7C5CFC),
                quizEnabled, { quizEnabled = it }) { showQuizSetup = true }

            Text("可同时开启多个条件，满足任一即可解锁",
                fontSize = 11.sp, color = Neutral400,
                modifier = Modifier.padding(top = 16.dp))

            Spacer()

            Button(
                onClick = { /* 保存条件 */ },
                modifier = Modifier.fillMaxWidth().height(48.dp),
                shape = RoundedCornerShape(12.dp),
                colors = ButtonDefaults.buttonColors(
                    backgroundColor = PrimaryColor,
                    contentColor = White
                )
            ) {
                Text("保存条件", fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun ConditionCard(
    icon: String,
    title: String,
    subtitle: String,
    color: Color,
    enabled: Boolean,
    onToggle: (Boolean) -> Unit,
    onTap: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        elevation = 4.dp,
        backgroundColor = White
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier.size(44.dp).clip(CircleShape).background(color.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Text(icon, fontSize = 20.sp)
            }
            Spacer(modifier = Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(title, fontSize = 16.sp, fontWeight = FontWeight.SemiBold, color = Neutral800)
                Text(subtitle, fontSize = 13.sp, color = Neutral600)
            }
            Switch(checked = enabled, onCheckedChange = onToggle,
                colors = SwitchDefaults.colors(checkedThumbColor = color))
        }
    }
}

// MARK: - 锁定消息卡片 (COND-007)

@Composable
fun LockedMessageCard(
    conditions: List<CondStatus>,
    onUnlock: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth().padding(16.dp),
        shape = RoundedCornerShape(8.dp),
        backgroundColor = Color(0xFFFFF8E1)
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Default.Lock, contentDescription = null, tint = Color(0xFFFF9500))
                Spacer(modifier = Modifier.width(8.dp))
                Text("对方设置了聊天条件", fontSize = 15.sp, fontWeight = FontWeight.SemiBold,
                    color = Color(0xFFFF9500))
            }

            Spacer(modifier = Modifier.height(12.dp))

            conditions.forEach { cond ->
                Row(verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.padding(vertical = 4.dp)) {
                    Box(modifier = Modifier.size(8.dp).clip(CircleShape)
                        .background(if (cond.met) Color(0xFF34C759) else Color(0xFFFF9500)))
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(cond.label, fontSize = 14.sp, color = Neutral800, modifier = Modifier.weight(1f))
                    Text(if (cond.met) "已满足" else "未满足", fontSize = 12.sp,
                        color = if (cond.met) Color(0xFF34C759) else Neutral400)
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            Text("满足任一条件即可解锁", fontSize = 11.sp, color = Neutral400)

            Spacer(modifier = Modifier.height(12.dp))

            Button(
                onClick = onUnlock,
                modifier = Modifier.width(140.dp).height(36.dp),
                shape = RoundedCornerShape(8.dp),
                colors = ButtonDefaults.buttonColors(
                    backgroundColor = PrimaryColor,
                    contentColor = White
                )
            ) {
                Text("开始解锁", fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
            }
        }
    }
}

data class CondStatus(val type: Int, val label: String, val met: Boolean)

// MARK: - 地图选点页 (LOC-002)

@Composable
fun LocationPickerScreen(
    friendId: String,
    onBack: () -> Unit
) {
    var radius by remember { mutableStateOf(500f) }

    Column(modifier = Modifier.fillMaxSize()) {
        // 地图占位
        Box(
            modifier = Modifier.fillMaxWidth().height(300.dp).background(Neutral100),
            contentAlignment = Alignment.Center
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Icon(Icons.Default.Map, "地图", tint = Color(0xFF20C0B0), modifier = Modifier.size(48.dp))
                Text("高德地图选点", color = Neutral600)
            }
        }

        Column(modifier = Modifier.padding(16.dp)) {
            Text("需要相距在 ${radius.toInt()}m 内", fontSize = 16.sp,
                fontWeight = FontWeight.SemiBold, color = Neutral800)
            Spacer(modifier = Modifier.height(12.dp))
            Slider(
                value = radius,
                onValueChange = { radius = it },
                valueRange = 200f..2000f,
                steps = 17,
                colors = SliderDefaults.colors(thumbColor = PrimaryColor, activeTrackColor = PrimaryColor)
            )
            Row(modifier = Modifier.fillMaxWidth()) {
                Text("200m", fontSize = 11.sp, color = Neutral400)
                Spacer()
                Text("2km", fontSize = 11.sp, color = Neutral400)
            }
        }

        Spacer()

        Button(
            onClick = { onBack() },
            modifier = Modifier.fillMaxWidth().height(48.dp).padding(horizontal = 16.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(backgroundColor = PrimaryColor, contentColor = White)
        ) {
            Text("确定", fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
        }
        Spacer(modifier = Modifier.height(32.dp))
    }
}

// MARK: - 答题设置页 (QUIZ-001)

@Composable
fun QuizSetupScreen(
    onBack: () -> Unit,
    onSave: (String, String, Int) -> Unit
) {
    var question by remember { mutableStateOf("") }
    var answer by remember { mutableStateOf("") }
    var maxTries by remember { mutableIntStateOf(3) }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Text("输入你的问题", fontSize = 13.sp, fontWeight = FontWeight.Medium, color = Neutral600)
        OutlinedTextField(
            value = question,
            onValueChange = { question = it.take(50) },
            placeholder = { Text("例如：我最喜欢的颜色是什么？") },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp)
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text("输入正确答案", fontSize = 13.sp, fontWeight = FontWeight.Medium, color = Neutral600)
        OutlinedTextField(
            value = answer,
            onValueChange = { answer = it.take(20) },
            placeholder = { Text("答案") },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp)
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text("允许尝试次数", fontSize = 13.sp, fontWeight = FontWeight.Medium, color = Neutral600)
        Row(modifier = Modifier.padding(top = 8.dp)) {
            listOf(1, 2, 3, 5).forEach { n ->
                FilterChip(
                    selected = maxTries == n,
                    onClick = { maxTries = n },
                    label = { Text("${n}次") },
                    modifier = Modifier.padding(end = 8.dp),
                    colors = FilterChipDefaults.filterChipColors(
                        selectedContainerColor = PrimaryColor.copy(alpha = 0.2f),
                        selectedLabelColor = PrimaryColor
                    )
                )
            }
        }

        Spacer()

        Button(
            onClick = { onSave(question, answer, maxTries) },
            enabled = question.isNotBlank() && answer.isNotBlank(),
            modifier = Modifier.fillMaxWidth().height(48.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(
                backgroundColor = if (question.isNotBlank() && answer.isNotBlank()) PrimaryColor else Neutral100,
                contentColor = if (question.isNotBlank() && answer.isNotBlank()) White else Neutral400
            )
        ) {
            Text("保存题目", fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
        }

        Spacer(modifier = Modifier.height(32.dp))
    }
}
