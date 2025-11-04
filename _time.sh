#!/usr/bin/bash

# GLOBAL VARS
SOLIDTIME_URL=${SOLIDTIME_URL:="https://chrono.woodpeqr.net"}
solidtime_api_url="$SOLIDTIME_URL/api/v1"
SOLIDTIME_ORG=${SOLIDTIME_ORG:="Stiffy's Organization"}
SOLIDTIME_PROJECT=${SOLIDTIME_PROJECT:="Work"}

work_hours_per_day=8

# LIBS
source "$(dirname $(realpath "$0"))/_utils.sh"

# LOCAL VARS
solidtime_user_id=
solidtime_org_id=
solidtime_project_id=

# FLAGS
time_offline=
time_from=30
time_start=
time_live=

format_seconds() {
    local total_s=$1
    local hours=$((total_s / 3600))
    local minutes=$(((total_s % 3600) / 60))
    local seconds=$((total_s % 60))
    stdout "${hours}h ${minutes}m ${seconds}s"
}

solidtime_get() {
    local endpoint=$1
    shift
    local url="$solidtime_api_url/$endpoint"

    if [[ -n $time_offline ]]; then
        error "offline mode, but still tried to make request. not user error"
    fi 

    if [[ -z $SOLIDTIME_API_KEY ]]; then
        error "\$SOLIDTIME_API_KEY env var is not set!"        
    fi

    local query
    query=$(printf "%s&" "$@")
    query="${query%&}"

    [[ -n "$query" ]] && url="$url?$query"

    res=$(curl \
        -H "Authorization: Bearer $SOLIDTIME_API_KEY" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -L \
        "$url")
    debug "$url =>" "$(echo "$res" | jq .)"
    stdout $res 
}

solidtime_assert_user_id(){
    if [[ -z "$solidtime_user_id" ]]; then
        solidtime_user_id=$(solidtime_get "/users/me" | jq -r .data.id)
    fi
}

solidtime_assert_org_id(){
    if [[ -z "$solidtime_org_id" ]]; then
        solidtime_org_id=$(solidtime_get "/users/me/memberships" | \
            jq -r --arg name "$SOLIDTIME_ORG" '.data[] | select(.organization.name == $name) | .organization.id')
    fi
}

solidtime_assert_project_id() {
    solidtime_assert_org_id
    if [[ -z "$solidtime_project_id" ]]; then
        solidtime_project_id=$(solidtime_get "/organizations/$solidtime_org_id/projects" | \
            jq -r --arg name "$SOLIDTIME_PROJECT" '.data[] | select(.name == $name) | .id')
    fi
}

solidtime_get_shift_stats() {
   if [[ -n $time_offline ]]; then
       log "running in offline mode, skipping fetching shift stats"
       return 0
   fi

   solidtime_assert_user_id
   solidtime_assert_org_id
   solidtime_assert_project_id

   local start="$(date -uv-${time_from}d +%Y-%m-%dT%H:%M:%SZ)"
   local end="$(date -uv+1 +%Y-%m-%dT%H:%M:%SZ)"

   local data="$(solidtime_get "/organizations/$solidtime_org_id/time-entries/aggregate" \
       "start=$start" \
       "end=$end" \
       "project_ids[]=$solidtime_project_id" \
       "user_id=$solidtime_user_id" \
       "group=day" \
       "active=false")"

   local seconds=$(echo "$data" | jq -r .data.seconds)
   local overtime=$((seconds - ($(echo "$data" | jq '.data.grouped_data | length') * 8 * 3600)))
   
   stdout $seconds $overtime
}

solidtime_get_current_shift_info() {
    if [[ -n "$time_offline" ]] && [[ -z "$time_start" ]]; then
        error "--offline requires --start to be passed"
        exit 1 
    fi

    solidtime_assert_user_id
    solidtime_assert_org_id
    solidtime_assert_project_id

    local start="$(date -uv-1d +%Y-%m-%dT%H:%M:%SZ)" 
    local end="$(date -uv+1d +%Y-%m-%dT%H:%M:%SZ)"

    local data="$(solidtime_get "/organizations/$solidtime_org_id/time-entries" \
        "start=$start" \
        "project_ids[]=$solidtime_project_id" \
        "user_id=$solidtime_user_id" \
        "active=true"
        )"

    stdout $data

}
