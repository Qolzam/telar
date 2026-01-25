#!/bin/bash

# ==============================================================================
# THE GUARDIAN PROTOCOL - FORENSIC STRESS TEST
# ==============================================================================

API_URL="http://localhost:9066/api/v1/analyze/content"
# API Key for authentication (use AI_ENGINE_INTERNAL_API_KEY or INTERNAL_API_KEY)
AUTH_TOKEN="${AI_ENGINE_INTERNAL_API_KEY:-${INTERNAL_API_KEY:-}}"

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# STATS
TOTAL=0
PASSED=0
FAILED=0

echo -e "${BLUE}====================================================================${NC}"
echo -e "${BLUE}STARTING FORENSIC VALIDATION${NC}"
echo -e "${BLUE}Target: ${API_URL}${NC}"
echo -e "${BLUE}====================================================================${NC}\n"

run_test() {
    ((TOTAL++))
    category=$1
    input=$2
    expect_flag=$3
    # We now expect specific "Reasons" (e.g., "Violence/Threat", "Personal Insult", "High Toxicity")
    expect_reason_part=$4

    json_input=$(jq -n --arg t "$input" '{"content": $t}')
    
    start_time=$(date +%s%N)
    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d "$json_input")
    duration=$(( ($(date +%s%N) - $start_time) / 1000000 ))

    got_flag=$(echo "$response" | jq -r '.is_flagged')
    got_reason=$(echo "$response" | jq -r '.flag_reason // "none"')
    got_model=$(echo "$response" | jq -r '.model_used // "unknown"')

    status="${GREEN}PASS${NC}"
    
    if [ "$got_flag" != "$expect_flag" ]; then
        status="${RED}FAIL (Flag Mismatch: expected $expect_flag, got $got_flag)${NC}"
    elif [ ! -z "$expect_reason_part" ] && [ "$expect_flag" = "true" ]; then
        # Only check reason if we expected a flag and reason_part is specified
        if [[ "$got_reason" != *"$expect_reason_part"* ]]; then
             status="${RED}FAIL (Reason Mismatch: expected '$expect_reason_part' in reason, got '$got_reason')${NC}"
        fi
    fi

    if [[ "$status" == *"${GREEN}PASS${NC}"* ]]; then
        ((PASSED++))
    else
        ((FAILED++))
    fi

    printf "%-15s | %-30s | %-6s | %-25s | %-20s | %-4s ms | %b\n" \
        "$category" "${input:0:28}..." "$got_flag" "$got_reason" "$got_model" "$duration" "$status"
    
    if [[ "$status" == *"${RED}FAIL"* ]]; then
        echo -e "    ${RED}Full Response: $response${NC}"
    fi
}

printf "%-15s | %-30s | %-6s | %-25s | %-20s | %-4s | %s\n" "Category" "Input" "Flag" "Reason" "Model" "Time" "Status"
echo "--------------------------------------------------------------------------------------------------------------------------------------------------------"

# ==============================================================================
# 1. FALSE POSITIVE CHECK (The "Mondays" Test)
# Expect: Safe. ONNX sees "Toxic" but not "Severe/Insult/Threat", so it passes. LLM Approves.
# ==============================================================================
echo -e "\n${YELLOW}--- FALSE POSITIVE CHECK (Must Be Safe) ---${NC}"
run_test "FalsePos" "I hate Mondays. Coffee is garbage." "false" ""

# ==============================================================================
# 2. THREAT DETECTION (High Priority)
# Expect: Flagged by ONNX as Violence/Threat
# ==============================================================================
echo -e "\n${YELLOW}--- THREAT DETECTION (High Priority) ---${NC}"
run_test "Threat" "I will break your legs." "true" "Violence"
run_test "Threat" "I will kill you." "true" "Violence"
run_test "Threat" "You deserve to die." "true" "Violence"

# ==============================================================================
# 3. INSULT DETECTION
# Expect: Flagged by L2 (keyword) OR L3 (ONNX) as Personal Insult
# Note: L2 may catch "idiot" before L3 runs - both are correct
# ==============================================================================
echo -e "\n${YELLOW}--- INSULT DETECTION ---${NC}"
run_test "Insult" "You are a stupid idiot." "true" ""  # Accept L2 keyword OR L3 ONNX
run_test "Insult" "Shut up you stupid loser." "true" "Insult"
run_test "Insult" "You are disgusting filth." "true" "Insult"

# ==============================================================================
# 4. HATE SPEECH / IDENTITY HATE
# Expect: Flagged by ONNX as Hate Speech OR High Toxicity (both are valid)
# Note: Model may score as "toxic" rather than "identity_hate" - both are correct flags
# ==============================================================================
echo -e "\n${YELLOW}--- HATE SPEECH DETECTION ---${NC}"
run_test "Hate" "You people are disgusting." "true" ""  # Accept "Hate Speech" OR "High Toxicity"
run_test "Hate" "Nobody loves you, die." "true" ""

# ==============================================================================
# 5. L2 HEURISTICS (Speed Check)
# Expect: Flagged by Keyword Filter
# ==============================================================================
echo -e "\n${YELLOW}--- L2 HEURISTICS (Keyword Filter) ---${NC}"
run_test "L2-Heuristic" "Join my crypto pump group" "true" "keyword"
run_test "L2-Heuristic" "Get free money now 100x gains" "true" "keyword"

# ==============================================================================
# 6. L4 LLM FALLBACK (Nuance)
# ONNX might miss this subtle threat, LLM should catch it.
# ==============================================================================
echo -e "\n${YELLOW}--- L4 LLM FALLBACK (Nuance Detection) ---${NC}"
run_test "L4-Nuance" "I hope you step on a lego." "true" ""
run_test "L4-Nuance" "Great job breaking production, genius." "true" ""
run_test "L4-Nuance" "You should try drinking bleach." "true" ""

# ==============================================================================
# 7. SAFE CONTENT (Additional False Positive Checks)
# ==============================================================================
echo -e "\n${YELLOW}--- SAFE CONTENT (False Positive Prevention) ---${NC}"
run_test "Safety-Check" "This process killed the server." "false" ""
run_test "Safety-Check" "I need to execute the child process." "false" ""
run_test "Safety-Check" "I violently disagree with your architectural choice." "false" ""
run_test "Safety-Check" "This coffee tastes like garbage." "false" ""
run_test "Safety-Check" "I am dying of laughter." "false" ""

# ==============================================================================
# FINAL REPORT
# ==============================================================================
echo -e "\n${BLUE}====================================================================${NC}"
echo -e "${BLUE}TEST COMPLETE${NC}"
echo -e "Total: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
    echo -e "${RED}VERDICT: SYSTEM NEEDS TUNING${NC}"
    exit 1
else
    echo -e "${GREEN}VERDICT: SYSTEM ENTERPRISE READY${NC}"
    exit 0
fi
