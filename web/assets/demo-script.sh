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
echo "+---------------+-----------+-------------------+---------------+"
echo "| INSTANCE TYPE |  REGION   | AVAILABILITY ZONE | SPOT PRICE/HR |"
echo "+---------------+-----------+-------------------+---------------+"
pause 0.2
echo "| t3.nano       | us-east-1 | us-east-1b        | \$0.0015       |"
pause 0.2
echo "| t3.micro      | us-east-1 | us-east-1a        | \$0.0029       |"
pause 0.2
echo "| t3.small      | us-east-1 | us-east-1c        | \$0.0062       |"
pause 0.2
echo "| t3.medium     | us-east-2 | us-east-2a        | \$0.0124       |"
pause 0.2
echo "| t3.large      | us-west-2 | us-west-2b        | \$0.0248       |"
echo "+---------------+-----------+-------------------+---------------+"
echo
pause 0.3
echo "💰 Spot Instance Summary:"
pause 0.2
echo "   Instance Types: 5"
pause 0.2
echo "   Average Spot Price: \$0.0096 per hour"
echo
pause 2

# Demo 2: spawn launch with TTL and name
echo -e "${BOLD}# Launch instance with name and idle detection${RESET}"
pause 1
type_command "spawn launch --name dev-box --idle-timeout 30m --ttl 4h us-east-1b t3.nano"
pause 0.5

echo
echo -e "${BOLD}🍄 Launching spore 'dev-box'...${RESET}"
pause 0.8
echo -e "   ${CYAN}Type:${RESET} t3.nano (2 vCPUs, 0.5 GB RAM)"
pause 0.3
echo -e "   ${CYAN}Region:${RESET} us-east-1b"
pause 0.3
echo -e "   ${CYAN}Cost:${RESET} \$0.0015/hr"
pause 0.3
echo -e "   ${CYAN}TTL:${RESET} 4h (auto-terminates)"
pause 0.3
echo -e "   ${CYAN}Idle Detection:${RESET} Hibernates after 30m idle"
pause 0.8
echo
echo -e "${YELLOW}⏳ Waiting for instance to start...${RESET}"
pause 1.5
echo -e "${GREEN}✓ Spore 'dev-box' running (i-0abc1234def5678)${RESET}"
pause 0.5
echo -e "${GREEN}✓ SSH: ssh ec2-user@3.84.123.45${RESET}"
pause 0.5
echo -e "${GREEN}✓ Idle detection active${RESET}"
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

# Demo 4: Idle detection and hibernation
echo -e "${BOLD}# Check status after 30 minutes idle${RESET}"
pause 1
type_command "spawn status dev-box"
pause 0.5

echo
echo -e "${CYAN}Spore:${RESET} dev-box (i-0abc1234def5678)"
pause 0.3
echo -e "${CYAN}Status:${RESET} hibernated"
pause 0.3
echo -e "${CYAN}Idle Time:${RESET} 31 minutes"
pause 0.3
echo -e "${CYAN}Cost Savings:${RESET} \$0.015/hr → \$0.001/hr (93% reduction)"
pause 0.5
echo
echo -e "${YELLOW}💤 Instance hibernated automatically after 30m idle${RESET}"
echo
pause 2

echo -e "${BOLD}# Wake and connect using spawn ssh${RESET}"
pause 1
type_command "spawn ssh dev-box"
pause 0.5

echo
echo -e "${YELLOW}⏳ Waking spore 'dev-box' from hibernation...${RESET}"
pause 1.5
echo -e "${GREEN}✓ Spore resumed in 45 seconds${RESET}"
pause 0.5
echo -e "${GREEN}✓ Connecting via SSH...${RESET}"
pause 0.5
echo
echo -e "${CYAN}Connected to dev-box (i-0abc1234def5678)${RESET}"
pause 0.5
echo -e "${CYAN}Resumed from hibernation - session restored${RESET}"
pause 0.5
echo
type_command "exit"
pause 0.3
echo
echo -e "${CYAN}Connection to dev-box closed.${RESET}"
echo
pause 2

# Demo 5: List and terminate
echo -e "${BOLD}# List running spores${RESET}"
pause 1
type_command "spawn list"
pause 0.5

echo
echo -e "${BOLD}Name       Instance ID          Type      Region      Status    TTL${RESET}"
echo "──────────────────────────────────────────────────────────────────────────"
pause 0.3
echo "dev-box    i-0abc1234def5678    t3.nano   us-east-1b  running   3h 57m"
pause 0.5
echo
echo -e "${GREEN}✓ 1 spore running${RESET}"
echo
pause 1.5

echo -e "${BOLD}# Terminate by name${RESET}"
pause 1
type_command "spawn terminate dev-box"
pause 0.5

echo
echo -e "${YELLOW}⏳ Terminating spore 'dev-box'...${RESET}"
pause 1
echo -e "${GREEN}✓ Spore 'dev-box' (i-0abc1234def5678) terminated${RESET}"
echo -e "${GREEN}✓ Total runtime: 4 minutes${RESET}"
echo -e "${GREEN}✓ Total cost: \$0.0001${RESET}"
echo
pause 2

# Demo 6: Quick command tip
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
