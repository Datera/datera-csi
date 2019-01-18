#!/bin/bash

KUBECTL="kubectl"
POD_REGEX="csi-(provisioner|node)-"

function genstr()
{
    cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 6 | head -n 1
}

SAVE_STR=$(genstr)
SAVE_DIR=/tmp/csi-logs-${SAVE_STR}
ARCHIVE=/tmp/csi-logs-${SAVE_STR}.tar.gz

function check_external_dependencies()
{
    #Make sure these executables exist in PATH
    local execs="${KUBECTL} grep awk tar"
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
    ${KUBECTL} describe pods --namespace kube-system ${pod} > ${SAVE_DIR}/${pod}_describe.txt
    for c in ${containers}
    do
        echo "[INFO] Saving container logfile: ${c}"
        ${KUBECTL} logs --namespace kube-system ${pod} ${c} > ${SAVE_DIR}/${c}.txt
    done
}

function collect_logs()
{
    echo "[INFO] Collecting CSI logs"
    mkdir -p ${SAVE_DIR}
    cat /etc/*-release > ${SAVE_DIR}/os_release.txt
    ${KUBECTL} version > ${SAVE_DIR}/kubectl_version.txt
    local pods=$(${KUBECTL} get pods --namespace kube-system | grep -E "${POD_REGEX}" | awk '{print $1}')
    for pod in ${pods}
    do
        echo "[INFO] Collecting for pod: ${pod}"
        collect_pod_logs ${pod}
    done
    echo "[INFO] Creating archive: ${ARCHIVE}"
    tar cvfz ${ARCHIVE} ${SAVE_DIR}
}

function usage()
{
    echo "
Usage: $0 [-k KUBECTL -hs]
-h Print Help
-s Skip dependency check
-k KUBECTL Use non-standard kubectl
-p POD_REGEX Regex (grep -E compatible) to match pods for log collection
"; exit 1;
}

OPT_S=false
while getopts ":hsk:p:" option
do
    case "${option}"
    in
        s) OPT_S=true
          ;;
        k) KUBECTL=${OPTARG}
          ;;
        p) POD_REGEX=${OPTARG}
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
