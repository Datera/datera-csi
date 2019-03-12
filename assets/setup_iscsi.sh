#!/bin/bash

IFILE="https://github.com/Datera/datera-csi/releases/download/v1.0.3/iscsi-recv.linux.x86-64"
DIR="/var/datera"
SFILE="/var/datera/iscsi-recv"
SCSI_SHA1="809e0816f6a294c68c07a71f7c7206f4ac165d9b"
SOCK="/var/datera/csi-iscsi.sock"

SCSI_SERVICE="[Unit]
Description = iscsi-recv container to host iscsiadm adapter service

[Service]
ExecStart = /var/datera/iscsi-recv -addr unix:///${SOCK}

[Install]
WantedBy = multi-user.target"

SCSI_SERVICE_NAME="iscsi-recv.service"
SCSI_SERVICE_FILE="/lib/systemd/system/${SCSI_SERVICE_NAME}"

function check_external_dependencies()
{
    #Make sure these executables exist in PATH
    local execs="curl sha1sum"
    echo "[INFO] Dependency checking"

    for bx in ${execs}
    do
            type -p "${bx}" &>/dev/null
            if [ $? != 0 ] ; then
                    echo "Error: Can't find  ${bx} in PATH=${PATH}"
                    exit 1
            fi
    done
}

function get_and_start_iscsi_recv()
{
    echo "[INFO] Downloading iscsi-recv"
    mkdir -p ${DIR}
    if ! curl -f -L --silent "${IFILE}" > "${SFILE}"
    then
        echo "[ERROR] Downloading iscsi-recv binary from ${IFILE}"
        exit 1
    fi

    echo "[INFO] Verifying checksum"
    local foundsum=$(sha1sum ${SFILE} | awk '{print $1}')
    if [[ ${foundsum} != "${SCSI_SHA1}" ]]
    then
        echo "[ERROR] Checksum of downloaded iscsi-recv does not match [${foundsum}] != [${SCSI_SHA1}]"
        exit 1
    fi

    echo "[INFO] Changing file permissions"
    chmod +x "${SFILE}"

    echo "[INFO] Registering iscsi-recv service"
    echo "${SCSI_SERVICE}" > "${SCSI_SERVICE_FILE}"
    if [ ! -e "${SCSI_SERVICE_FILE}" ]
    then
        echo "[ERROR] Failed to create '${SCSI_SERVICE_FILE}'.  Exiting"
        exit 1
    fi
    systemctl enable "${SCSI_SERVICE_NAME}"
    if [[ $? != 0 ]]
    then
        echo "[ERROR] Failed to register iscsi-recv service"
        exit 1
    fi

    echo "[INFO] Starting iscsi-recv service"
    systemctl start "${SCSI_SERVICE_NAME}"
    if [[ $? != 0 ]]
    then
        echo "[ERROR] Failed to start iscsi-recv service"
        exit 1
    fi

    sleep 1

    echo "[INFO] Verifying service started correctly"
    ps -ef | grep "${SFILE}" | grep -v "grep"
    if [[ $? != 0 ]]
    then
        echo "[ERROR] Failed to verify iscsi-recv service is running"
    fi
}

function uninstall()
{
    echo "[INFO] Stopping service"
    systemctl stop "${SCSI_SERVICE_NAME}"
    echo "[INFO] Disabling service"
    systemctl disable "${SCSI_SERVICE_NAME}"
    echo "[INFO] Removing service file"
    rm -- "${SCSI_SERVICE_FILE}"
    echo "[INFO] Reloading/Resetting systemctl"
    systemctl daemon-reload
    systemctl reset-failed
    echo "[INFO] Removing binary"
    rm -- "${SFILE}" "${SOCK}"
}

function usage()
{
    echo "Usage: $0 [-hu]
-h  Print Usage
-u  Uninstall iscsi-recv" 1>&2; exit 1;
}

while getopts ":uh" option
do
    case "${option}"
    in
        u) OPT_U=true
          ;;
        \?) echo "Invalid Option: -${OPTARG}" >&2; exit 1
          ;;
        *) usage
          ;;
    esac
done

if [[ ${OPT_U} == true ]]
then
    uninstall
else
    check_external_dependencies
    get_and_start_iscsi_recv
fi
