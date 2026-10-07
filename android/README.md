# FunChat Android

timo胶囊（FunChat）Android 客户端工程。本目录补齐了此前缺失的工程化文件，
使 `android/` 成为一个可被 Android Studio / Gradle 直接构建的标准项目。

## 环境要求

| 项 | 版本 | 说明 |
|---|---|---|
| Android Studio | Koala 2024.1.1+ | 建议最新稳定版 |
| JDK | 17 | AGP 8.5 强制要求 |
| compileSdk / targetSdk | 34 | |
| minSdk | 26 | BUG-011：Android 8.0 以下启动崩溃 |
| Gradle | 8.7 | 已随仓库提供 wrapper，无需单独安装 |
| AGP / Kotlin | 8.5.0 / 1.9.24 | |
| Compose | BOM 2024.06.00（Material 2） | 与现有 `ui/*` 代码一致 |

## 打开方式

**File → Open → 选择本目录 `android/`**（不是仓库根目录），然后等待 Gradle Sync 完成。

> 因为 `local.properties` 未提交，首次打开时 Studio 会提示配置 SDK 路径，按向导指到本机 Android SDK 即可。

## 后端联调

1. 起后端（仓库根目录）：
   ```bash
   docker compose up -d
   curl http://localhost:8080/health
   ```
2. `data/api/APIClient.kt` 的 `BASE_URL` 已是 `http://10.0.2.2:8080/api/v1`，
   **模拟器访问宿主机必须用 `10.0.2.2`**，不能用 `localhost`。
3. 开发期已在 Manifest 打开 `android:usesCleartextTraffic="true"` 以允许明文 HTTP。
   **上线前必须移除该属性并切换 HTTPS。**

## 命令行构建

```bash
cd android
./gradlew assembleDebug          # macOS / Linux
gradlew.bat assembleDebug        # Windows
```

产物：`android/app/build/outputs/apk/debug/app-debug.apk`

仅做编译校验（更快）：

```bash
./gradlew :app:compileDebugKotlin
```

## 目录结构

```
android/
├── settings.gradle.kts
├── build.gradle.kts              # 顶层：只声明插件版本
├── gradle.properties
├── gradlew / gradlew.bat
├── gradle/wrapper/
└── app/
    ├── build.gradle.kts          # 依赖清单
    └── src/main/
        ├── AndroidManifest.xml
        ├── java/com/funchat/
        │   ├── MainActivity.kt        # 新增：入口 + 页面状态机
        │   ├── data/api/APIClient.kt
        │   ├── ui/auth/
        │   ├── ui/chat/
        │   └── ui/friends/
        └── res/
            ├── values/themes.xml
            ├── values/strings.xml
            └── drawable/ic_launcher.xml
```

## 已知待办（按事项）

| 事项 | 内容 |
|---|---|
| `rz7mgu` | 防截图 `FLAG_SECURE`；定位 / 步数（ACTIVITY_RECOGNITION）运行时权限申请 |
| `rz7mgu` | 高德定位 SDK、系统计步 Sensor 接入 |
| V1.1 | 微信登录、COS 媒体上传 |

- 启动图标目前是占位矢量图（`res/drawable/ic_launcher.xml`），等设计输出后替换为正式 mipmap + adaptive icon。
- 页面间靠回调导航，未引入 `navigation-compose`；页面再增加时建议迁移。
