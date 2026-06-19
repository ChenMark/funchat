-- funchat V1.0 数据库迁移脚本
-- 数据库: MySQL 8.0
-- 引擎: InnoDB
-- 字符集: utf8mb4

CREATE DATABASE IF NOT EXISTS funchat
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE funchat;

-- ============================================================
-- 1. 用户表
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id              VARCHAR(32)   NOT NULL PRIMARY KEY COMMENT '用户ID (雪花ID)',
    nickname        VARCHAR(20)   NOT NULL DEFAULT '' COMMENT '昵称',
    avatar          VARCHAR(500)  NOT NULL DEFAULT '' COMMENT '头像URL',
    phone           VARCHAR(11)   NOT NULL DEFAULT '' COMMENT '手机号',
    wechat_union_id VARCHAR(64)   NOT NULL DEFAULT '' COMMENT '微信UnionID',
    apple_user_id   VARCHAR(128)  NOT NULL DEFAULT '' COMMENT 'Apple User ID',
    gender          TINYINT       NOT NULL DEFAULT 0 COMMENT '性别: 0=未知 1=男 2=女',
    birthday        VARCHAR(10)   NOT NULL DEFAULT '' COMMENT '生日 YYYY-MM-DD',
    status          TINYINT       NOT NULL DEFAULT 1 COMMENT '状态: 1=正常 2=冻结 3=注销',
    created_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX idx_phone (phone),
    UNIQUE INDEX idx_wechat_union_id (wechat_union_id),
    UNIQUE INDEX idx_apple_user_id (apple_user_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- ============================================================
-- 2. 好友关系表
-- ============================================================
CREATE TABLE IF NOT EXISTS friendships (
    id          BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     VARCHAR(32)   NOT NULL COMMENT '用户ID',
    friend_id   VARCHAR(32)   NOT NULL COMMENT '好友ID',
    status      TINYINT       NOT NULL DEFAULT 1 COMMENT '状态: 1=正常 2=已删除',
    source      VARCHAR(20)   NOT NULL DEFAULT 'search' COMMENT '来源: search/qrcode/recommend',
    created_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '添加时间',
    updated_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_friend (user_id, friend_id),
    INDEX idx_friend_id (friend_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友关系表';

-- ============================================================
-- 3. 好友申请表
-- ============================================================
CREATE TABLE IF NOT EXISTS friend_requests (
    id          BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
    from_user_id VARCHAR(32)  NOT NULL COMMENT '申请发起方',
    to_user_id  VARCHAR(32)   NOT NULL COMMENT '申请接收方',
    message     VARCHAR(50)   NOT NULL DEFAULT '' COMMENT '验证消息',
    status      TINYINT       NOT NULL DEFAULT 0 COMMENT '状态: 0=待处理 1=已同意 2=已拒绝',
    created_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
    updated_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_to_user_status (to_user_id, status),
    INDEX idx_from_user (from_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友申请表';

-- ============================================================
-- 4. 消息表
-- ============================================================
CREATE TABLE IF NOT EXISTS messages (
    id            BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    from_user_id  VARCHAR(32) NOT NULL COMMENT '发送方',
    to_user_id    VARCHAR(32) NOT NULL COMMENT '接收方',
    msg_type      TINYINT     NOT NULL COMMENT '消息类型: 1=文本 2=图片 3=系统消息',
    content       TEXT        COMMENT '消息内容',
    is_read       TINYINT     NOT NULL DEFAULT 0 COMMENT '已读: 0=未读 1=已读',
    is_burn       TINYINT     NOT NULL DEFAULT 0 COMMENT '阅后即焚: 0=普通 1=焚毁',
    burn_duration INT         NOT NULL DEFAULT 0 COMMENT '焚毁倒计时(秒)',
    created_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发送时间',
    INDEX idx_conversation (from_user_id, to_user_id, created_at),
    INDEX idx_to_user_read (to_user_id, is_read)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息表';

-- ============================================================
-- 5. 解锁条件配置表
-- ============================================================
CREATE TABLE IF NOT EXISTS condition_configs (
    id         BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id    VARCHAR(32) NOT NULL COMMENT '设置者',
    cond_type  TINYINT     NOT NULL COMMENT '条件类型: 1=定位 2=步数 3=答题',
    is_enabled TINYINT     NOT NULL DEFAULT 1 COMMENT '启用: 1=是 0=否',
    params     TEXT        COMMENT '条件参数 (JSON)',
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_cond (user_id, cond_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='解锁条件配置表';

-- ============================================================
-- 6. 解锁记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS unlock_records (
    id         BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id    VARCHAR(32) NOT NULL COMMENT '解锁者',
    target_id  VARCHAR(32) NOT NULL COMMENT '被解锁对象',
    cond_type  TINYINT     NOT NULL COMMENT '条件类型',
    result     TINYINT     NOT NULL DEFAULT 0 COMMENT '结果: 0=验证中 1=成功 2=失败',
    detail     TEXT        COMMENT '验证详情',
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '解锁时间',
    INDEX idx_user_target (user_id, target_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='解锁记录表';

-- ============================================================
-- 7. 阅后即焚记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS burn_records (
    id         BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    message_id BIGINT      NOT NULL COMMENT '消息ID',
    user_id    VARCHAR(32) NOT NULL COMMENT '查看者',
    read_at    DATETIME    COMMENT '已读时间',
    burn_at    DATETIME    COMMENT '焚毁时间',
    status     TINYINT     NOT NULL DEFAULT 0 COMMENT '状态: 0=未读 1=已读(倒计时) 2=已焚毁',
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_message_id (message_id),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='阅后即焚记录表';
