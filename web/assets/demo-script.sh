#!/bin/bash
# Demo script for spore.host asciicinema recording

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RESET='\033[0m'
BOLD='\033[1m'

# Function to simulate typing
type_command() {
    local cmd="$1"
    echo -ne "${GREEN}$ ${RESET}"
    for (( i=0; i<${#cmd}; i++ )); do
        echo -n "${cmd:$i:1}"
        sleep 0.05
    done
    echo
    sleep 0.3
}

# Function to wait
pause() {
    sleep "${1:-1}"
}

clear

# Show header
echo -e "${BOLD}${BLUE}╔════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${BLUE}║                   spore.host Demo                          ║${RESET}"
echo -e "${BOLD}${BLUE}║      Launch EC2 in 2 minutes. Auto-terminate. Free.       ║${RESET}"
echo -e "${BOLD}${BLUE}╚════════════════════════════════════════════════════════════╝${RESET}"
echo
pause 2

# Demo 1: truffle spot search
echo -e "${BOLD}# Find the cheapest spot instances${RESET}"
pause 1
type_command "truffle spot \"t3.*\" --sort-by-price --limit 5"
pause 0.5

echo
echo -e "${BOLD}${CYAN}┌─────────────────────────────────────────────────────────────┐${RESET}"
echo -e "${BOLD}${CYAN}│ 🔍 Truffle - EC2 Spot Instance Search                      │${RESET}"
echo -e "${BOLD}${CYAN}└─────────────────────────────────────────────────────────────┘${RESET}"
echo
pause 0.3

echo -e "${BOLD}Instance Type    vCPUs  Memory   Spot Price   Region      Savings${RESET}"
echo "─────────────────────────────────────────────────────────────────"
pause 0.2
echo "t3.nano          2      0.5 GB   \$0.0016/hr   us-east-1   95%"
pause 0.2
echo "t3.micro         2      1.0 GB   \$0.0031/hr   us-east-1   94%"
pause 0.2
echo "t3.small         2      2.0 GB   \$0.0063/hr   us-east-1   93%"
pause 0.2
echo "t3.medium        2      4.0 GB   \$0.0125/hr   us-east-2   92%"
pause 0.2
echo "t3.large         2      8.0 GB   \$0.0250/hr   us-west-2   91%"
echo
pause 0.3
echo -e "${GREEN}✓ Found 5 instances with spot capacity available${RESET}"
echo
pause 2

# Demo 2: spawn launch with TTL
echo -e "${BOLD}# Launch the cheapest instance with 4-hour TTL${RESET}"
pause 1
type_command "truffle spot \"t3.*\" --sort-by-price --pick-first | spawn --ttl 4h"
pause 0.5

echo
echo -e "${BOLD}🍄 Launching instance...${RESET}"
pause 0.8
echo -e "   ${CYAN}Type:${RESET} t3.nano (2 vCPUs, 0.5 GB RAM)"
pause 0.3
echo -e "   ${CYAN}Region:${RESET} us-east-1"
pause 0.3
echo -e "   ${CYAN}Cost:${RESET} \$0.0016/hr (~\$0.01/day)"
pause 0.3
echo -e "   ${CYAN}TTL:${RESET} 4h (auto-terminates)"
pause 0.8
echo
echo -e "${YELLOW}⏳ Waiting for instance to start...${RESET}"
pause 1.5
echo -e "${GREEN}✓ Instance i-0abc1234def5678 running${RESET}"
pause 0.5
echo -e "${GREEN}✓ SSH: ssh ec2-user@3.84.123.45${RESET}"
pause 0.5
echo -e "${GREEN}✓ Auto-terminates in 3h 59m${RESET}"
echo
pause 2

# Demo 3: Quick command tip
echo -e "${BOLD}# Pro tip: Launch with one command${RESET}"
pause 1
type_command "spawn --type t3.micro --region us-west-2 --ttl 8h"
pause 0.5

echo
echo -e "${BOLD}🚀 Quick launch mode...${RESET}"
pause 0.8
echo -e "${GREEN}✓ Launched in 32 seconds${RESET}"
echo -e "${GREEN}✓ IP: 54.184.89.123${RESET}"
echo -e "${GREEN}✓ Cost: \$0.0031/hr${RESET}"
echo
pause 2

# Closing
echo
echo -e "${BOLD}${BLUE}════════════════════════════════════════════════════════════${RESET}"
echo -e "${BOLD}${CYAN}  Get started: https://spore.host${RESET}"
echo -e "${BOLD}${CYAN}  Free & open source • Auto-terminate • Zero surprise bills${RESET}"
echo -e "${BOLD}${BLUE}════════════════════════════════════════════════════════════${RESET}"
echo
pause 2
