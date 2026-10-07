// FunChat Android — 顶层构建脚本
// 只声明插件版本（apply false），具体 apply 在各 module 的 build.gradle.kts 中

plugins {
    id("com.android.application") version "8.5.0" apply false
    id("org.jetbrains.kotlin.android") version "1.9.24" apply false
}
