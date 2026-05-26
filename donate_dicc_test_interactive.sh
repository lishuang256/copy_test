#!/bin/bash

SCRIPTDIR="$(dirname $(readlink --canonicalize ${BASH_SOURCE}))"
FPC_PATH="${SCRIPTDIR}/.."
FABRIC_SCRIPTDIR="${FPC_PATH}/fabric/bin/"

: ${FABRIC_CFG_PATH:="${SCRIPTDIR}/config"}

. ${FABRIC_SCRIPTDIR}/lib/common_utils.sh
. ${FABRIC_SCRIPTDIR}/lib/common_ledger.sh

CC_ID=donate_dicc_test
CC_PATH=${FPC_PATH}/samples/chaincode/donate/donate_dicc/_build/lib/
CC_LANG=fpc-c
CC_VER="$(cat ${CC_PATH}/mrenclave)"
CC_SEQ="1"
CC_EP="OR('SampleOrg.member')" # note that we use .member as NodeOUs is disabled with the crypto material used in the integration tests.

NUM_FAILURES=0

pause_for_enter() {
    local message="$1"
    if [[ -z "${message}" ]]; then
        message="按回车继续..."
    fi
    read -r -p "${message}"
}

donate_dosc_test() {
    PKG=/tmp/${CC_ID}.tar.gz

    try ${PEER_CMD} lifecycle chaincode package --lang ${CC_LANG} --label ${CC_ID} --path ${CC_PATH} ${PKG}
    try ${PEER_CMD} lifecycle chaincode install ${PKG}

    PKG_ID=$(${PEER_CMD} lifecycle chaincode queryinstalled | awk "/Package ID: ${CC_ID}/{print}" | sed -n 's/^Package ID: //; s/, Label:.*$//;p')

    try ${PEER_CMD} lifecycle chaincode approveformyorg -o ${ORDERER_ADDR} -C ${CHAN_ID} --package-id ${PKG_ID} --name ${CC_ID} --version ${CC_VER} --sequence ${CC_SEQ} --signature-policy ${CC_EP}

    try ${PEER_CMD} lifecycle chaincode checkcommitreadiness -C ${CHAN_ID} --name ${CC_ID} --version ${CC_VER} --sequence ${CC_SEQ} --signature-policy ${CC_EP}

    try ${PEER_CMD} lifecycle chaincode commit -o ${ORDERER_ADDR} -C ${CHAN_ID} --name ${CC_ID} --version ${CC_VER} --sequence ${CC_SEQ} --signature-policy ${CC_EP}

    try ${PEER_CMD} lifecycle chaincode initEnclave -o ${ORDERER_ADDR} --peerAddresses "localhost:7051" --name ${CC_ID}

    try ${PEER_CMD} lifecycle chaincode querycommitted -C ${CHAN_ID}

    echo "2. 开始执行FPC交易"
    pause_for_enter "按回车开始执行FPC交易..."

    cat <<'CMD'
(1) 测试Distribute交易: ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Distribute", "Args": ["100", "100facemask", "10921244836505220497059200573276005902561255475336363492765253626011689884813", "96265766022768170794111142310884989615889545183642865508508811539477701719982", "1001", "1111", "2025-10-01T12:01:01"]}' --waitForEvent
CMD
    pause_for_enter "按回车执行第(1)条交易..."
    ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Distribute", "Args": ["100", "100facemask", "10921244836505220497059200573276005902561255475336363492765253626011689884813", "96265766022768170794111142310884989615889545183642865508508811539477701719982", "1001", "1111", "2025-10-01T12:01:01"]}' --waitForEvent
    pause_for_enter "第(1)条交易已返回结果，按回车继续..."

    cat <<'CMD'
(2) 测试Refund交易, 结果为False: ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Refund", "Args": ["101facemask", "100000facemask10921244836505220497059200573276005902561255475336363492765253626011689884813962657660227681707941111423108849896158895451836428655085088115394777017199822025-10-01T12:01:01"]}' --waitForEvent
CMD
    pause_for_enter "按回车执行第(2)条交易..."
    ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Refund", "Args": ["101facemask", "100000facemask10921244836505220497059200573276005902561255475336363492765253626011689884813962657660227681707941111423108849896158895451836428655085088115394777017199822025-10-01T12:01:01"]}' --waitForEvent
    pause_for_enter "第(2)条交易已返回结果，按回车继续..."

    cat <<'CMD'
(3) 测试Refund交易, 结果为True: ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Refund", "Args": ["101facemask", "100100facemask10921244836505220497059200573276005902561255475336363492765253626011689884813962657660227681707941111423108849896158895451836428655085088115394777017199822025-10-01T12:01:01"]}' --waitForEvent
CMD
    pause_for_enter "按回车执行第(3)条交易..."
    ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Refund", "Args": ["101facemask", "100100facemask10921244836505220497059200573276005902561255475336363492765253626011689884813962657660227681707941111423108849896158895451836428655085088115394777017199822025-10-01T12:01:01"]}' --waitForEvent
    pause_for_enter "第(3)条交易已返回结果，按回车继续..."

    cat <<'CMD'
(4) 测试Audit交易: ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Audit", "Args": []}' --waitForEvent
CMD
    pause_for_enter "按回车执行第(4)条交易..."
    ${PEER_CMD} chaincode invoke -o ${ORDERER_ADDR} -C ${CHAN_ID} -n ${CC_ID} -c '{"Function":"Audit", "Args": []}' --waitForEvent
    pause_for_enter "第(4)条交易已返回结果，按回车继续..."
}

# 1. prepare
para
say "Preparing Donate_DiCC Test ..."
# - clean up relevant docker images
docker_clean ${ERCC_ID}

para
say "Run donate_dicc test"

echo "1. 初始化FPC网络"
pause_for_enter "按回车开始执行 ledger_init..."
ledger_init

say "- donate_dicc test"
donate_dosc_test

echo "3. 关闭Fabric网络"
pause_for_enter "按回车执行 ledger_shutdown..."
ledger_shutdown

exit 0
