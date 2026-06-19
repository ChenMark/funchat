#!/bin/bash
# FunChat 全栈回归测试脚本
# 覆盖: Phase 0 (health) + Phase 1 (Auth + Friends) + Phase 2 (Chat + WS) + Phase 3 (Conditions)
# 生成日期: 2026-06-19

set -e

BASE="http://localhost:8080/api/v1"
PASS=0
FAIL=0
RESULTS=""

# 颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

test_case() {
    local name="$1"
    local method="$2"
    local url="$3"
    local auth="$4"
    local data="$5"
    local expected_code="$6"

    local auth_header=""
    if [ -n "$auth" ]; then
        auth_header="-H \"Authorization: Bearer $auth\""
    fi

    local data_arg=""
    if [ -n "$data" ]; then
        data_arg="-d '$data'"
    fi

    local cmd="curl -s -X $method \"$url\" -H \"Content-Type: application/json\" $auth_header $data_arg"
    local response=$(eval "$cmd" 2>/dev/null)
    local actual_code=$(echo "$response" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code', -1))" 2>/dev/null || echo "-1")

    if [ "$actual_code" == "$expected_code" ]; then
        PASS=$((PASS + 1))
        RESULTS+="${GREEN}PASS${NC} | $name | code=$actual_code\n"
        echo -e "${GREEN}✓${NC} $name"
    else
        FAIL=$((FAIL + 1))
        RESULTS+="${RED}FAIL${NC} | $name | expected=$expected_code actual=$actual_code\n"
        echo -e "${RED}✗${NC} $name (expected=$expected_code actual=$actual_code)"
        echo "  Response: $response"
    fi
}

echo "==============================================="
echo "  FunChat 全栈回归测试"
echo "  覆盖 Phase 0-3"
echo "==============================================="
echo ""

# ==========================================
# Phase 0: 基础设施
# ==========================================
echo "--- Phase 0: 基础设施 ---"

# 健康检查
test_case "Health Check" "GET" "http://localhost:8080/health" "" "" "0"

echo ""

# ==========================================
# Phase 1: 用户体系 + 好友模块
# ==========================================
echo "--- Phase 1: 用户体系 + 好友模块 ---"

# 1.1 发送验证码
SEND_A=$(curl -s -X POST "$BASE/auth/send-code" -H "Content-Type: application/json" -d '{"phone":"13800000001"}')
CODE_A=$(echo "$SEND_A" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['code'])")
test_case "发送验证码 (用户A)" "POST" "$BASE/auth/send-code" "" '{"phone":"13800000001"}' "0"

# 1.2 手机号格式错误
test_case "手机号格式错误" "POST" "$BASE/auth/send-code" "" '{"phone":"123"}' "40000"

# 1.3 校验验证码
VERIFY_A=$(curl -s -X POST "$BASE/auth/verify-code" -H "Content-Type: application/json" -d "{\"phone\":\"13800000001\",\"code\":\"$CODE_A\"}")
CT_A=$(echo "$VERIFY_A" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['code_token'])")
test_case "校验验证码 (用户A)" "POST" "$BASE/auth/verify-code" "" "{\"phone\":\"13800000001\",\"code\":\"$CODE_A\"}" "0"

# 1.4 验证码错误
test_case "验证码错误" "POST" "$BASE/auth/verify-code" "" '{"phone":"13800000001","code":"000000"}' "40011"

# 1.5 注册
REG_A=$(curl -s -X POST "$BASE/auth/register" -H "Content-Type: application/json" -d "{\"phone\":\"13800000001\",\"code_token\":\"$CT_A\",\"nickname\":\"测试用户A\"}")
TA=$(echo "$REG_A" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
test_case "注册用户A" "POST" "$BASE/auth/register" "" "{\"phone\":\"13800000001\",\"code_token\":\"$CT_A\",\"nickname\":\"测试用户A\"}" "0"

# 1.6 创建用户B
SEND_B=$(curl -s -X POST "$BASE/auth/send-code" -H "Content-Type: application/json" -d '{"phone":"13800000002"}')
CODE_B=$(echo "$SEND_B" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['code'])")
VERIFY_B=$(curl -s -X POST "$BASE/auth/verify-code" -H "Content-Type: application/json" -d "{\"phone\":\"13800000002\",\"code\":\"$CODE_B\"}")
CT_B=$(echo "$VERIFY_B" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['code_token'])")
REG_B=$(curl -s -X POST "$BASE/auth/register" -H "Content-Type: application/json" -d "{\"phone\":\"13800000002\",\"code_token\":\"$CT_B\",\"nickname\":\"测试用户B\"}")
TB=$(echo "$REG_B" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
RT_B=$(echo "$REG_B" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['refresh_token'])")
test_case "注册用户B" "POST" "$BASE/auth/register" "" "{\"phone\":\"13800000002\",\"code_token\":\"$CT_B\",\"nickname\":\"测试用户B\"}" "0"

# 1.7 登录
SEND_C=$(curl -s -X POST "$BASE/auth/send-code" -H "Content-Type: application/json" -d '{"phone":"13800000001"}')
CODE_C=$(echo "$SEND_C" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['code'])")
VERIFY_C=$(curl -s -X POST "$BASE/auth/verify-code" -H "Content-Type: application/json" -d "{\"phone\":\"13800000001\",\"code\":\"$CODE_C\"}")
CT_C=$(echo "$VERIFY_C" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['code_token'])")
test_case "手机号登录" "POST" "$BASE/auth/login" "" "{\"phone\":\"13800000001\",\"code_token\":\"$CT_C\"}" "0"

# 1.8 Token 刷新
test_case "Token刷新" "POST" "$BASE/auth/refresh" "$TA" "{\"refresh_token\":\"$RT_B\"}" "0"

# 1.9 微信登录
test_case "微信登录(未绑定)" "POST" "$BASE/auth/wechat/login" "" '{"code":"wx_test_12345"}' "0"

# 1.10 Apple 登录
test_case "Apple登录(未绑定)" "POST" "$BASE/auth/apple/login" "" '{"identity_token":"apple_test_67890"}' "0"

# 1.11 未授权访问
test_case "未授权访问拦截" "GET" "$BASE/friends" "" "" "40100"

# 1.12 好友搜索
test_case "好友搜索" "GET" "$BASE/friends/search?keyword=小明" "$TA" "" "0"

# 1.13 发送好友申请
test_case "发送好友申请" "POST" "$BASE/friends/request" "$TA" '{"target_user_id":"u_13800000002","message":"你好"}' "0"

# 1.14 重复申请
test_case "重复申请拦截" "POST" "$BASE/friends/request" "$TA" '{"target_user_id":"u_13800000002"}' "40900"

# 1.15 添加自己
test_case "添加自己拦截" "POST" "$BASE/friends/request" "$TA" '{"target_user_id":"u_13800000001"}' "40030"

# 1.16 查看申请
test_case "查看申请列表" "GET" "$BASE/friends/requests" "$TB" "" "0"

# 1.17 同意申请
test_case "同意好友申请" "POST" "$BASE/friends/accept" "$TB" '{"request_id":1}' "0"

# 1.18 好友列表
test_case "好友列表(A)" "GET" "$BASE/friends" "$TA" "" "0"
test_case "好友列表(B)" "GET" "$BASE/friends" "$TB" "" "0"

# 1.19 删除好友
test_case "删除好友" "DELETE" "$BASE/friends/u_13800000002" "$TA" "" "0"

echo ""

# ==========================================
# Phase 2: 聊天核心
# ==========================================
echo "--- Phase 2: 聊天核心 ---"

# 重新加好友
curl -s -X POST "$BASE/friends/request" -H "Content-Type: application/json" -H "Authorization: Bearer $TA" -d '{"target_user_id":"u_13800000002"}' > /dev/null
curl -s -X POST "$BASE/friends/accept" -H "Content-Type: application/json" -H "Authorization: Bearer $TB" -d '{"request_id":2}' > /dev/null

# 2.1 发送消息
MSG_RESP=$(curl -s -X POST "$BASE/messages/send" -H "Content-Type: application/json" -H "Authorization: Bearer $TA" -d '{"to_user_id":"u_13800000002","msg_type":1,"content":"你好，这是测试消息！"}')
MSG_ID=$(echo "$MSG_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['msg_id'])")
test_case "发送文本消息" "POST" "$BASE/messages/send" "$TA" '{"to_user_id":"u_13800000002","msg_type":1,"content":"你好，这是测试消息！"}' "0"

# 2.2 消息历史
test_case "消息历史" "GET" "$BASE/messages/history?friend_id=u_13800000002&page=1&size=20" "$TA" "" "0"

# 2.3 聊天列表
test_case "聊天列表" "GET" "$BASE/conversations" "$TA" "" "0"

# 2.4 在线状态
test_case "在线状态查询" "GET" "$BASE/online/u_13800000002" "$TA" "" "0"

echo ""

# ==========================================
# Phase 3: 解锁条件
# ==========================================
echo "--- Phase 3: 解锁条件 ---"

# 3.1 条件消息发送
test_case "发送条件消息" "POST" "$BASE/messages/send-conditional" "$TA" '{"to_user_id":"u_13800000002","content":"条件消息","cond_types":[1,2,3]}' "0"

# 3.2 设置条件
test_case "设置解锁条件" "POST" "$BASE/conditions/set" "$TA" '{"cond_type":1,"is_enabled":1,"params":"{\"radius\":500}"}' "0"

# 3.3 获取条件
test_case "获取条件配置" "GET" "$BASE/conditions/u_13800000001" "$TA" "" "0"

# 3.4 定位校验
test_case "定位校验(范围内)" "POST" "$BASE/conditions/verify-location" "$TA" '{"target_user_id":"u_13800000002","lat":22.5431,"lng":113.9266}' "0"

# 3.5 虚拟定位检测(正常)
test_case "虚拟定位检测(正常)" "POST" "$BASE/conditions/detect-fake-location" "$TA" '{"lat":22.5431,"lng":113.9266,"accuracy":15,"is_mock":false}' "0"

# 3.6 虚拟定位检测(异常)
test_case "虚拟定位检测(异常)" "POST" "$BASE/conditions/detect-fake-location" "$TA" '{"lat":22.5431,"lng":113.9266,"accuracy":0,"is_mock":true}' "0"

# 3.7 步数校验(达标)
test_case "步数校验(达标)" "POST" "$BASE/conditions/verify-steps" "$TA" '{"target_user_id":"u_13800000002","steps":9500}' "0"

# 3.8 步数防作弊
test_case "步数防作弊(正常)" "POST" "$BASE/conditions/detect-step-cheating" "$TA" '{"steps":9500,"start_time":1700000000,"end_time":1700086400}' "0"

# 3.9 设置题目
test_case "设置答题" "POST" "$BASE/conditions/set-quiz" "$TA" '{"question":"测试问题","answer":"答案","max_tries":3}' "0"

# 3.10 获取题目
test_case "获取题目(无答案)" "GET" "$BASE/conditions/quiz/1" "$TA" "" "0"

# 3.11 答题正确
test_case "答题(正确)" "POST" "$BASE/conditions/verify-quiz" "$TA" '{"quiz_id":1,"answer":"答案"}' "0"

# 3.12 答题错误
test_case "答题(错误)" "POST" "$BASE/conditions/verify-quiz" "$TA" '{"quiz_id":1,"answer":"错误答案"}' "0"

# 3.13 答题次数用完(模拟)
test_case "答题次数用完" "POST" "$BASE/conditions/verify-quiz" "$TA" '{"quiz_id":1,"answer":"答案"}' "40050"

echo ""

# ==========================================
# 结果汇总
# ==========================================
echo "==============================================="
echo "  测试结果汇总"
echo "==============================================="
echo -e "通过: ${GREEN}$PASS${NC}"
echo -e "失败: ${RED}$FAIL${NC}"
echo -e "总计: $((PASS + FAIL))"
echo ""
echo -e "$RESULTS"

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}全部测试通过！${NC}"
    exit 0
else
    echo -e "${RED}存在 $FAIL 个失败测试${NC}"
    exit 1
fi
