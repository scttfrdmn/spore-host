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

# Demo 1: truffle spot search
echo -e "${BOLD}# Find the cheapest spot instances${RESET}"
pause 1
type_command "truffle spot \"t3.*\" --sort-by-price --limit 5"
pause 0.5

echo
pause 0.5
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

# Demo 3: SSH into instance
echo -e "${BOLD}# SSH into the instance${RESET}"
pause 1
type_command "ssh ec2-user@3.84.123.45"
pause 0.5

echo
echo -e "${CYAN}Connected to instance i-0abc1234def5678${RESET}"
pause 0.5
echo -e "${CYAN}Amazon Linux 2023${RESET}"
pause 0.5
echo
type_command "uptime"
pause 0.3
echo " 16:42:15 up 2 min,  1 user,  load average: 0.00, 0.00, 0.00"
pause 0.5
type_command "exit"
pause 0.3
echo
echo -e "${CYAN}Connection to 3.84.123.45 closed.${RESET}"
echo
pause 1.5

# Demo 4: List and terminate
echo -e "${BOLD}# List running instances${RESET}"
pause 1
type_command "spawn list"
pause 0.5

echo
echo -e "${BOLD}Instance ID           Type      Region      Status    TTL Remaining${RESET}"
echo "─────────────────────────────────────────────────────────────────────"
pause 0.3
echo "i-0abc1234def5678    t3.nano   us-east-1   running   3h 57m"
pause 0.5
echo
echo -e "${GREEN}✓ 1 instance running${RESET}"
echo
pause 1.5

echo -e "${BOLD}# Terminate instance${RESET}"
pause 1
type_command "spawn terminate i-0abc1234def5678"
pause 0.5

echo
echo -e "${YELLOW}⏳ Terminating instance...${RESET}"
pause 1
echo -e "${GREEN}✓ Instance i-0abc1234def5678 terminated${RESET}"
echo -e "${GREEN}✓ Total runtime: 4 minutes${RESET}"
echo -e "${GREEN}✓ Total cost: \$0.0001${RESET}"
echo
pause 2

# Demo 5: Quick command tip
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
