#!/bin/bash

# Color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'
DIM='\033[2m'

# Default selections (Documentation disabled by default)
selected=(1 1 0)
current=0
services=("Frontend" "CE" "Documentation")
scripts=("./scripts/start_dev_frontend.sh" "./scripts/start_dev_ce.sh" "./scripts/start_dev_doc.sh")

# Hide cursor
tput civis

# Cleanup function
cleanup() {
    tput cnorm  # Show cursor
    echo -e "\n"
}
trap cleanup EXIT

# Function to display menu
display_menu() {
    clear
    echo -e "${BLUE}${BOLD}=== CE Dev Launcher ===${NC}\n"
    echo -e "${DIM}Use ↑/↓ to navigate, Space to toggle, Enter to start, Q to quit${NC}\n"
    
    for i in "${!services[@]}"; do
        if [ $i -eq $current ]; then
            # Highlighted item
            echo -en "${CYAN}> "
        else
            echo -n "  "
        fi
        
        # Checkbox
        if [ ${selected[$i]} -eq 1 ]; then
            echo -en "${GREEN}[✓]${NC} "
        else
            echo -en "[ ] "
        fi
        
        # Service name
        echo -e "${services[$i]}"
    done
    
    echo -e "\n${DIM}────────────────────────────────────${NC}"
    echo -e "${YELLOW}[A]${NC} Select All  ${YELLOW}[N]${NC} Select None  ${GREEN}[Enter]${NC} Start  ${YELLOW}[Q]${NC} Quit"
}

# Read single character
read_key() {
    local key
    IFS= read -rsn1 key
    
    # Handle escape sequences (arrow keys)
    if [[ $key == $'\x1b' ]]; then
        read -rsn2 key
        case $key in
            '[A') echo "up" ;;
            '[B') echo "down" ;;
        esac
    else
        case $key in
            ' ') echo "space" ;;
            '') echo "enter" ;;
            q|Q) echo "quit" ;;
            a|A) echo "all" ;;
            n|N) echo "none" ;;
        esac
    fi
}

# Main loop
while true; do
    display_menu
    
    key=$(read_key)
    
    case $key in
        up)
            ((current--))
            if [ $current -lt 0 ]; then
                current=$((${#services[@]} - 1))
            fi
            ;;
        down)
            ((current++))
            if [ $current -ge ${#services[@]} ]; then
                current=0
            fi
            ;;
        space)
            selected[$current]=$((1 - ${selected[$current]}))
            ;;
        all)
            selected=(1 1 1)
            ;;
        none)
            selected=(0 0 0)
            ;;
        enter)
            break
            ;;
        quit)
            echo -e "\n${YELLOW}Exiting...${NC}"
            exit 0
            ;;
    esac
done

# Build the command based on selections
run_services=()
run_names=()

for i in "${!services[@]}"; do
    if [ ${selected[$i]} -eq 1 ]; then
        run_services+=("${scripts[$i]}")
        run_names+=("${services[$i],,}")  # Convert to lowercase
    fi
done

# Check if any services are selected
if [ ${#run_services[@]} -eq 0 ]; then
    clear
    echo -e "${YELLOW}No services selected. Exiting.${NC}"
    exit 0
fi

# Join array elements for concurrently
names_joined=$(IFS=,; echo "${run_names[*]}")
services_joined=$(IFS=" "; echo "${run_services[*]}")

clear
echo -e "${GREEN}${BOLD}Starting services: ${names_joined}${NC}\n"

# Run with concurrently
pnpm dlx concurrently -n "$names_joined" $services_joined