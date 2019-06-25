#!/bin/bash

KUBECTL="kubectl"
POD_REGEX="csi-(provisioner|node)-"
RSYNC="rsync://rts7.daterainc.com:/dumps/"
USERNAME="root"

function genstr()
{
    cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 6 | head -n 1
}

SAVE_STR=$(genstr)
SAVE_DIR=/tmp/csi-logs-$(hostname)-${SAVE_STR}
ARCHIVE=/tmp/csi-logs-$(hostname)-${SAVE_STR}.tar.gz

function check_external_dependencies()
{
    #Make sure these executables exist in PATH
    local execs="${KUBECTL} grep awk tar rsync sshpass"
    echo "[INFO] Dependency checking"

    for bx in ${execs}
    do
            type -p "${bx}" &>/dev/null
            if [ $? != 0 ] ; then
                    echo "Error: Can't find  ${bx} in PATH=${PATH}"
                    exit 1
            fi
    done

    # Check that grep supports pcre
    grep --help | grep "perl-regexp"
    if [[ $? != 0 ]]
    then
        echo "[ERROR] grep does not support PCRE (-P option)"
        exit 1
    fi
}

function collect_pod_logs()
{
    local pod=$1
    local containers=$(${KUBECTL} logs --namespace kube-system ${pod} 2>&1 | grep -Po "\[\K.*(?=\])")
    mkdir -p ${SAVE_DIR}/${pod}/
    ${KUBECTL} describe pods --namespace kube-system ${pod} > ${SAVE_DIR}/${pod}/describe.txt
    for c in ${containers}
    do
        if [[ ${OPT_I} == true ]] && [[ ${c} == *"liveness"* ]]
        then
            echo "[INFO] Skipping liveness probe log collection"
            continue
        fi
        echo "[INFO] Saving container logfile: ${c}"
        ${KUBECTL} logs --namespace kube-system ${pod} ${c} > ${SAVE_DIR}/${pod}/${c}.txt
        ${KUBECTL} logs --namespace kube-system --previous=true ${pod} ${c} > ${SAVE_DIR}/${pod}/${c}-previous.txt
    done
}

function collect_logs()
{
    echo "[INFO] Collecting CSI logs"
    mkdir -p ${SAVE_DIR}
    # OS information
    cat /etc/*-release > ${SAVE_DIR}/os_release.txt
    # Kubectl Version
    ${KUBECTL} version > ${SAVE_DIR}/kubectl_version.txt
    # Local node messages log
    mkdir -p ${SAVE_DIR}/$(hostname)
    cat /var/log/messages > ${SAVE_DIR}/$(hostname)/messages
    dmesg > ${SAVE_DIR}/$(hostname)/dmesg
    # iscsi-recv logs
    journalctl -u iscsi-recv.service > ${SAVE_DIR}/$(hostname)/iscsi-recv
    # Find relevant pods
    local pods=$(${KUBECTL} get pods --namespace kube-system | grep -E "${POD_REGEX}" | awk '{print $1}')
    # Collect pod logs
    for pod in ${pods}
    do
        echo "[INFO] Collecting for pod: ${pod}"
        collect_pod_logs ${pod}
    done
}

function collect_remote_logs() {
    echo "[INFO] Collecting logs from remote nodes"
    if [[ ${HOST_IPS} == "" ]]
    then
        HOST_IPS=$(kubectl describe nodes | grep InternalIP | awk '{print $2}' | xargs)
        HOSTNAMES=$(kubectl describe nodes | grep Hostname | awk '{print $2}' | xargs)
    fi
    if [[ ${OPT_R} == true ]]
    then
        mv ~/.ssh/known_hosts ~/.ssh/known_hosts.old
    fi
    echo "[INFO] HOST_IPS: ${HOST_IPS}"
    local arrhip=(${HOST_IPS})
    local arrhn=(${HOSTNAMES})
    for i in ${!arrhip[@]}
    do
        local ip=${arrhip[$i]}
        local hn=${arrhn[$i]}
        echo "[INFO] Collecting logs from ${ip}, ${hn}"
        mkdir -p ${SAVE_DIR}/${ip}
        sshpass -p "${PASSWORD}" scp -o "StrictHostKeyChecking=no" "${USERNAME}@${ip}:/var/log/messages*" ${SAVE_DIR}/${ip}/
        sshpass -p "${PASSWORD}" ssh -o "StrictHostKeyChecking=no" ${USERNAME}@${ip} journalctl -u iscsi-recv.service > ${SAVE_DIR}/${ip}/iscsi-recv
        sshpass -p "${PASSWORD}" ssh -o "StrictHostKeyChecking=no" ${USERNAME}@${ip} dmesg > ${SAVE_DIR}/${ip}/dmesg
        sshpass -p "${PASSWORD}" ssh -o "StrictHostKeyChecking=no" ${USERNAME}@${ip} last > ${SAVE_DIR}/${ip}/last
        touch ${SAVE_DIR}/${ip}/${hn}
    done
}

function create_archive() {
    echo "[INFO] Creating archive: ${ARCHIVE}"
    # Tar archive
    tar cvfz ${ARCHIVE} -C ${SAVE_DIR} . > /dev/null 2>&1
    if [[ $? != 0 ]]
    then
        echo "[ERROR] Failed to create archive"
        exit 1
    fi
    echo "[INFO] Archive size: $(ls -hl ${ARCHIVE} | awk '{print $5}')"
}

function upload_logs()
{
    echo "[INFO] Uploading logs to rts7.daterainc.com/dumps/$(basename ${ARCHIVE})"
    rsync -a ${ARCHIVE} ${RSYNC}
    if [[ $? != 0 ]]
    then
        echo "[ERROR] Failed to upload archive"
        exit 1
    fi
}

function usage()
{
    echo "
Datera CSI pod log collect script.

This script will iterate through all relevant CSI pods and download their logs
into a tarball archive located in /tmp/

Usage: $0 [-k KUBECTL -p POD_REGEX -hs]
-h Print Help (optional)
-s Skip dependency check (optional)
-u Upload logs to /dumps (optional)
-i Skip liveness probe log collect (optional, useful if hanging on liveness
   probe log collection
-k KUBECTL Use non-standard kubectl (optional, default: kubectl)
-p POD_REGEX Regex (optional, grep -E compatible) to match pods for log collection
-n USERNAME Username for kubernetes nodes (default: root)
-l PASSWORD Password for kubernetes nodes (optional unless collecting system logs)
-b HOST_IPS Comma delimited list of ip addresses (optional, provide if log
            collect can't determine them from kubectl describe nodes)
-r Remove known_hosts file (backed up to known_hosts.old.  Use this if ssh into
   remote nodes is failing
"; exit 1;
}

OPT_S=false
while getopts ":hsurik:p:n:l:" option
do
    case "${option}"
    in
        s) OPT_S=true
          ;;
        u) OPT_U=true
          ;;
        k) KUBECTL=${OPTARG}
          ;;
        p) POD_REGEX=${OPTARG}
          ;;
        n) USERNAME=${OPTARG}
          ;;
        l) PASSWORD=${OPTARG}
          ;;
        b) HOST_IPS=$(echo "${OPTARG}" | sed 's/,/ /g')
          ;;
        r) OPT_R=true
          ;;
        i) OPT_I=true
          ;;
        \?) echo "Invalid Option: -${OPTARG}" >&2; exit 1
          ;;
        *) usage
          ;;
    esac
done

if [[ ${OPT_S} == false ]]
then
    check_external_dependencies
fi

collect_logs

if [[ ${PASSWORD} != "" ]]
then
    collect_remote_logs
fi

create_archive

if [[ ${OPT_U} == true ]]
then
    upload_logs
fi
