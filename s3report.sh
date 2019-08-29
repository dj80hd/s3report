#!/usr/bin/env bash
set -o pipefail

_usage() {
  cat <<"EOF"
  s3report provides information about s3 buckets: Name, Creation Date, 
  Last Modified Date, Total Object Count, Total Size, and Size per Account
  
  It also lists the oldest (--count negative) or newest (--count positive) objects in the bucket.
  Default is 5 oldest

  --exclude and --include flags specify substrings that can filter the bucket list.
  Default is to include all buckets 
  
  Example: Report any bucket name containing substring 'accounting', including the 10 oldest objects
  s3report --count -10 --include accounting

  Example: Report all buckets except for accounting include newest objects:
  s3report --count 4 --exclude accounting
EOF
  exit 1
}

OPT_INCLUDE="" ; OPT_EXCLUDE="" ; OPT_COUNT=-5

# Process options and dependencies TODO: Check aws and jq are installed
_process_options() {
  while [[ $# -gt 0 ]]; do
    local key="$1"
    case "${key}" in
      --count)   OPT_COUNT="$2";        shift; shift;;
      --include) OPT_INCLUDE="$2";      shift; shift;;
      --exclude) OPT_EXCLUDE="$2";      shift; shift;;
      *) _usage;
    esac
  done
}

# e.g. convert 1234567 to 1.17 MiB
_bytes_to_human(){
  b=${1:-0}; d=''; s=0; S=(Bytes {K,M,G,T,E,P,Y,Z}iB)
  while ((b > 1024)); do
    d="$(printf ".%02d" $((b % 1024 * 100 / 1024)))"
    b=$((b / 1024))
    let s++
  done
  echo "$b$d ${S[$s]}"
}

# Get all bucket names filtered by the OPT_INCLUDE and OPT_EXCLUDE options
_bucket_names() {
  if [[ -n ${OPT_EXCLUDE} ]]; then
    aws s3api list-buckets | jq -rc '.Buckets[] |.Name' | grep "${OPT_INCLUDE}" | grep -v "{$OPT_EXCLUDE}"
  else
    aws s3api list-buckets | jq -rc '.Buckets[] |.Name' | grep "${OPT_INCLUDE}"
  fi
}

_main() {
  _process_options "$@"
  declare -a owner_sizes

  # Get JSON for all objects in buckets sorted by newest or oldest
  for b in $(_bucket_names); do
    if [ "${OPT_COUNT}" -lt "0" ]; then
      OPT_COUNT=${OPT_COUNT#-} 
      type="Oldest"
      objects_json=$(aws s3api list-objects --bucket ${b} --query "sort_by(Contents,&LastModified)")
    else
      type="Newest"
      objects_json=$(aws s3api list-objects --bucket ${b} --query "reverse(sort_by(Contents,&LastModified))")
    fi
   
    # Get total size per owner in the onwer_sizes hash
    owners=$(echo "${objects_json}" | jq -rc '.[] | .Owner.Id' | sort -u)
    for owner in ${owners}; do
      owner_size=$(echo "${objects_json}" | jq --arg owner ${owner} -rc '[.[] | select(.Owner.Id == $owner)] | map(.Size)|add ')
      owner_sizes[$owner]=${owner_size}
    done

    # Get all other bits of info.
    bucket_last_modified=$(echo "${objects_json}" | jq -rc '.[0] | .LastModified')
    bucket_size=$(echo "${objects_json}" | jq -rc 'map(.Size) | add')
    object_count=$(echo "${objects_json}" | jq 'length')
    objects=$(echo "${objects_json}" | jq --arg n "${OPT_COUNT}" -rc '.[:($n|tonumber)] | .[] | " " + .LastModified + " " + .Key + " " + (.Size |tostring)')
    bucket_creation_date=$(aws s3api list-buckets | jq --arg name ${b} -rc '.Buckets[] |select(.Name == $name) | .CreationDate') 

    printf "\nName: $b\n"
    printf "Created: ${bucket_creation_date}\n"
    printf "LastModified: ${bucket_last_modified}\n"
    printf "Object Count: ${object_count}\n"
    printf "TotalSize: $(_bytes_to_human ${bucket_size})\n"
    printf "${type} ${OPT_COUNT} Objects:\n"
    [[ -n ${objects} ]] && printf "${objects}\n"
    for owner in "${owners}"; do printf "Owner $owner Total Size: $(_bytes_to_human ${owner_sizes[$owner]})\n"; done
  done 
}

_main "$@"
