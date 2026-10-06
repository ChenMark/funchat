#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_BASE="${BASE_URL}/api/v1"
THIRD_PARTY_LOGIN_EXPECTED_CODE="${THIRD_PARTY_LOGIN_EXPECTED_CODE:-0}"
THIRD_PARTY_LOGIN_EXPECTED_HTTP="${THIRD_PARTY_LOGIN_EXPECTED_HTTP:-200}"

PASS=0
FAIL=0

LAST_BODY=""
LAST_HTTP=""
LAST_CODE=""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

timestamp="$(date +%s)"
PHONE_A="1388${timestamp: -7}"
PHONE_B="1399${timestamp: -7}"

TOKEN_A=""
TOKEN_B=""
REFRESH_A=""
REQUEST_ID=""
QUIZ_ID=""
BURN_MSG_ID=""

log_section() {
    echo
    echo "=== $1 ==="
}

api_call() {
    local method="$1"
    local path="$2"
    local token="${3:-}"
    local data="${4:-}"

    local args=(-sS -X "$method" "${API_BASE}${path}" -H "Content-Type: application/json")
    if [[ -n "$token" ]]; then
        args+=(-H "Authorization: Bearer ${token}")
    fi
    if [[ -n "$data" ]]; then
        args+=(-d "$data")
    fi

    local raw
    raw="$(curl "${args[@]}" -w $'\n%{http_code}')"
    LAST_BODY="$(printf '%s\n' "$raw" | sed '$d')"
    LAST_HTTP="$(printf '%s\n' "$raw" | tail -n 1)"
    LAST_CODE="$(printf '%s' "$LAST_BODY" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("code", -1))')"
}

record_pass() {
    PASS=$((PASS + 1))
    echo -e "${GREEN}PASS${NC} $1"
}

record_fail() {
    FAIL=$((FAIL + 1))
    echo -e "${RED}FAIL${NC} $1"
    echo "  HTTP: ${LAST_HTTP}"
    echo "  Body: ${LAST_BODY}"
}

assert_status() {
    local name="$1"
    local expected_http="$2"
    local expected_code="$3"

    if [[ "$LAST_HTTP" == "$expected_http" && "$LAST_CODE" == "$expected_code" ]]; then
        record_pass "$name"
    else
        record_fail "$name (expected http=${expected_http} code=${expected_code})"
    fi
}

extract_json() {
    local expr="$1"
    printf '%s' "$LAST_BODY" | python3 -c "import json,sys; data=json.load(sys.stdin); value=${expr}; print('' if value is None else value)"
}

assert_json_value() {
    local name="$1"
    local expr="$2"
    local expected="$3"
    local actual
    actual="$(extract_json "$expr")"

    if [[ "$actual" == "$expected" ]]; then
        record_pass "$name"
    else
        FAIL=$((FAIL + 1))
        echo -e "${RED}FAIL${NC} $name"
        echo "  expected: ${expected}"
        echo "  actual:   ${actual}"
        echo "  body:     ${LAST_BODY}"
    fi
}

echo "FunChat API regression smoke"
echo "BASE_URL=${BASE_URL}"
echo "PHONE_A=${PHONE_A}"
echo "PHONE_B=${PHONE_B}"

log_section "Health"
api_call GET "" "" ""
LAST_BODY="$(curl -sS "${BASE_URL}/health")"
LAST_HTTP="$(curl -sS -o /dev/null -w "%{http_code}" "${BASE_URL}/health")"
LAST_CODE="$(printf '%s' "$LAST_BODY" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("code", -1))')"
assert_status "health" "200" "0"

log_section "Auth"
api_call POST "/auth/send-code" "" "{\"phone\":\"${PHONE_A}\"}"
assert_status "send code A" "200" "0"
CODE_A="$(extract_json 'data["data"]["code"]')"

api_call POST "/auth/verify-code" "" "{\"phone\":\"${PHONE_A}\",\"code\":\"${CODE_A}\"}"
assert_status "verify code A" "200" "0"
CODE_TOKEN_A="$(extract_json 'data["data"]["code_token"]')"

api_call POST "/auth/register" "" "{\"phone\":\"${PHONE_A}\",\"code_token\":\"${CODE_TOKEN_A}\",\"nickname\":\"RegressionA\"}"
assert_status "register A" "201" "0"
TOKEN_A="$(extract_json 'data["data"]["access_token"]')"
REFRESH_A="$(extract_json 'data["data"]["refresh_token"]')"

api_call POST "/auth/send-code" "" "{\"phone\":\"${PHONE_B}\"}"
assert_status "send code B" "200" "0"
CODE_B="$(extract_json 'data["data"]["code"]')"

api_call POST "/auth/verify-code" "" "{\"phone\":\"${PHONE_B}\",\"code\":\"${CODE_B}\"}"
assert_status "verify code B" "200" "0"
CODE_TOKEN_B="$(extract_json 'data["data"]["code_token"]')"

api_call POST "/auth/register" "" "{\"phone\":\"${PHONE_B}\",\"code_token\":\"${CODE_TOKEN_B}\",\"nickname\":\"RegressionB\"}"
assert_status "register B" "201" "0"
TOKEN_B="$(extract_json 'data["data"]["access_token"]')"

api_call POST "/auth/send-code" "" "{\"phone\":\"${PHONE_A}\"}"
assert_status "send code A for login" "200" "0"
CODE_LOGIN_A="$(extract_json 'data["data"]["code"]')"

api_call POST "/auth/verify-code" "" "{\"phone\":\"${PHONE_A}\",\"code\":\"${CODE_LOGIN_A}\"}"
assert_status "verify code A for login" "200" "0"
CODE_TOKEN_LOGIN_A="$(extract_json 'data["data"]["code_token"]')"

api_call POST "/auth/login" "" "{\"phone\":\"${PHONE_A}\",\"code_token\":\"${CODE_TOKEN_LOGIN_A}\"}"
assert_status "login A" "200" "0"

api_call POST "/auth/refresh" "$TOKEN_A" "{\"refresh_token\":\"${REFRESH_A}\"}"
assert_status "refresh token A" "200" "0"

api_call POST "/auth/wechat/login" "" '{"code":"wx_test_code"}'
assert_status "wechat login" "${THIRD_PARTY_LOGIN_EXPECTED_HTTP}" "${THIRD_PARTY_LOGIN_EXPECTED_CODE}"
if [[ "${THIRD_PARTY_LOGIN_EXPECTED_CODE}" == "0" ]]; then
    assert_json_value "wechat login need_bind" 'data["data"]["need_bind"]' "True"
fi

api_call POST "/auth/apple/login" "" '{"identity_token":"apple_test_token"}'
assert_status "apple login" "${THIRD_PARTY_LOGIN_EXPECTED_HTTP}" "${THIRD_PARTY_LOGIN_EXPECTED_CODE}"

log_section "Friends"
api_call GET "/friends" "" ""
assert_status "friends unauthorized" "401" "40100"

api_call GET "/friends/search?keyword=${PHONE_B}" "$TOKEN_A" ""
assert_status "search friend by phone" "200" "0"

api_call POST "/friends/request" "$TOKEN_A" "{\"target_user_id\":\"u_${PHONE_B}\",\"message\":\"hello\"}"
assert_status "send friend request" "201" "0"
REQUEST_ID="$(extract_json 'data["data"]["request_id"]')"

api_call POST "/friends/request" "$TOKEN_A" "{\"target_user_id\":\"u_${PHONE_B}\"}"
assert_status "duplicate friend request blocked" "409" "40900"

api_call POST "/friends/request" "$TOKEN_A" "{\"target_user_id\":\"u_${PHONE_A}\"}"
assert_status "self friend request blocked" "400" "40030"

api_call GET "/friends/requests" "$TOKEN_B" ""
assert_status "list incoming requests" "200" "0"

api_call POST "/friends/accept" "$TOKEN_B" "{\"request_id\":${REQUEST_ID}}"
assert_status "accept friend request" "200" "0"

api_call GET "/friends" "$TOKEN_A" ""
assert_status "list friends A" "200" "0"

api_call GET "/friends" "$TOKEN_B" ""
assert_status "list friends B" "200" "0"

log_section "Messaging"
api_call POST "/messages/send" "$TOKEN_A" "{\"to_user_id\":\"u_${PHONE_B}\",\"msg_type\":1,\"content\":\"hello from regression\"}"
assert_status "send normal message" "201" "0"
MSG_ID="$(extract_json 'data["data"]["msg_id"]')"

api_call GET "/messages/history?friend_id=u_${PHONE_B}&page=1&size=20" "$TOKEN_A" ""
assert_status "message history" "200" "0"

api_call GET "/conversations" "$TOKEN_A" ""
assert_status "conversation list" "200" "0"

api_call POST "/messages/send-burn" "$TOKEN_A" "{\"to_user_id\":\"u_${PHONE_B}\",\"msg_type\":1,\"content\":\"burn after read\",\"duration\":5}"
assert_status "send burn message" "201" "0"
BURN_MSG_ID="$(extract_json 'data["data"]["msg_id"]')"

api_call GET "/burns/pending" "$TOKEN_B" ""
assert_status "list pending burns" "200" "0"

api_call POST "/messages/${BURN_MSG_ID}/burn-read" "$TOKEN_B" ""
assert_status "read burn message" "200" "0"
assert_json_value "burn read status" 'data["data"]["status"]' "reading"

api_call GET "/messages/${BURN_MSG_ID}/burn-status" "$TOKEN_A" ""
assert_status "burn status" "200" "0"

api_call POST "/messages/${BURN_MSG_ID}/burn-destroy" "$TOKEN_A" ""
assert_status "destroy burn message" "200" "0"

api_call POST "/messages/upload-image" "$TOKEN_A" ""
assert_status "upload image unavailable" "501" "50101"

api_call POST "/messages/upload-voice" "$TOKEN_A" ""
assert_status "upload voice unavailable" "501" "50101"

api_call DELETE "/friends/u_${PHONE_B}" "$TOKEN_A" ""
assert_status "delete friend" "200" "0"

api_call POST "/messages/send" "$TOKEN_A" "{\"to_user_id\":\"u_${PHONE_B}\",\"msg_type\":1,\"content\":\"should fail after delete\"}"
assert_status "send message blocked when not friends" "403" "40300"

api_call POST "/friends/request" "$TOKEN_A" "{\"target_user_id\":\"u_${PHONE_B}\",\"message\":\"re-add\"}"
assert_status "recreate friend request" "201" "0"
REQUEST_ID="$(extract_json 'data["data"]["request_id"]')"

api_call POST "/friends/accept" "$TOKEN_B" "{\"request_id\":${REQUEST_ID}}"
assert_status "accept recreated friend request" "200" "0"

log_section "Conditional Unlock (enabled in V1.0)"
api_call POST "/messages/send-conditional" "$TOKEN_A" "{\"to_user_id\":\"u_${PHONE_B}\",\"content\":\"blocked\",\"cond_types\":[1,2,3]}"
assert_status "conditional send" "201" "0"

api_call POST "/messages/${MSG_ID}/revoke" "$TOKEN_A" ""
assert_status "conditional revoke" "200" "0"

api_call GET "/messages/${MSG_ID}/condition-status" "$TOKEN_A" ""
assert_status "conditional status" "200" "0"

api_call POST "/conditions/verify-location" "$TOKEN_A" "{\"target_user_id\":\"u_${PHONE_B}\",\"lat\":22.5431,\"lng\":113.9266}"
assert_status "location verify" "200" "0"

api_call POST "/conditions/detect-fake-location" "$TOKEN_A" "{\"lat\":22.5431,\"lng\":113.9266,\"accuracy\":15,\"is_mock\":false}"
assert_status "fake location detect" "200" "0"

api_call POST "/conditions/verify-steps" "$TOKEN_A" "{\"target_user_id\":\"u_${PHONE_B}\",\"steps\":9500}"
assert_status "steps verify" "200" "0"

api_call POST "/conditions/detect-step-cheating" "$TOKEN_A" "{\"steps\":9500,\"start_time\":1700000000,\"end_time\":1700086400}"
assert_status "step cheating detect" "200" "0"

log_section "Supported Conditions"
api_call POST "/conditions/set" "$TOKEN_A" '{"cond_type":1,"is_enabled":1,"params":"{\"radius\":500}"}'
assert_status "set conditions" "201" "0"

api_call GET "/conditions/u_${PHONE_A}" "$TOKEN_A" ""
assert_status "get conditions" "200" "0"

api_call POST "/conditions/set-quiz" "$TOKEN_A" '{"question":"Q?","answer":"A","max_tries":3}'
assert_status "set quiz" "201" "0"
QUIZ_ID="$(extract_json 'data["data"]["quiz_id"]')"

api_call GET "/conditions/quiz/${QUIZ_ID}" "$TOKEN_A" ""
assert_status "get quiz" "200" "0"

api_call POST "/conditions/verify-quiz" "$TOKEN_A" "{\"quiz_id\":${QUIZ_ID},\"answer\":\"A\"}"
assert_status "verify quiz correct" "200" "0"
assert_json_value "quiz correct flag" 'data["data"]["correct"]' "True"

api_call POST "/conditions/verify-quiz" "$TOKEN_A" "{\"quiz_id\":${QUIZ_ID},\"answer\":\"wrong-1\"}"
assert_status "verify quiz wrong attempt 2" "200" "0"

api_call POST "/conditions/verify-quiz" "$TOKEN_A" "{\"quiz_id\":${QUIZ_ID},\"answer\":\"wrong-2\"}"
assert_status "verify quiz wrong attempt 3" "200" "0"

api_call POST "/conditions/verify-quiz" "$TOKEN_A" "{\"quiz_id\":${QUIZ_ID},\"answer\":\"blocked\"}"
assert_status "quiz max tries enforced" "429" "40050"

echo
echo "========================================"
echo "Passed: ${PASS}"
echo "Failed: ${FAIL}"
echo "========================================"

if [[ "$FAIL" -gt 0 ]]; then
    exit 1
fi
