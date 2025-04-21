#!/bin/bash
# Initialize associative array for storing max memory values in MiB
declare -A max_memory

while true; do
  # Use process substitution instead of a pipe to avoid subshell variable scope issues
  while read line; do
    name=$(echo "$line" | awk '{print $1}')
    mem_usage=$(echo "$line" | awk '{print $2}')

    # Extract the numeric value and unit (MiB or GiB)
    memory_value=$(echo "$mem_usage" | grep -o -E '[0-9]+(\.[0-9]+)?')
    unit=$(echo "$mem_usage" | grep -o -E '[A-Za-z]+')

    # Convert to MiB if the unit is GiB
    if [[ "$unit" == "GiB" ]]; then
      memory_mib=$(echo "$memory_value * 1024" | bc -l)
    else
      memory_mib=$memory_value
    fi

    # Round to 2 decimal places
    memory_mib=$(printf "%.2f" "$memory_mib")

    # Initialize max memory if not set
    if [[ -z ${max_memory[$name]} ]]; then
      max_memory[$name]=$memory_mib
    fi

    # Update max memory if current usage is higher
    if [[ $(echo "$memory_mib > ${max_memory[$name]}" | bc -l) -eq 1 ]]; then
      max_memory[$name]=$memory_mib
    fi

    # Display current and max memory - all in MiB
    echo "Container: $name - Current RAM: ${memory_mib} MiB - Max RAM: ${max_memory[$name]} MiB"
  done < <(docker stats --no-stream --format "{{.Name}}\t{{.MemUsage}}")

  sleep 0.1
done