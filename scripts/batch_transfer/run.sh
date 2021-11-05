#!/bin/sh
cd $(dirname $0)
echo "- 当前工作目录: $(pwd)"
# rm -rf ./keystore

# check if airdrop list empty
if [ -s 1_填写空投列表.csv ]; then
    echo "- 检查空投列表完成"
else
    echo "空投文件\"1_填写空投列表.csv\"为空，请添加空投列表"
    exit 1
fi

while [ "$address" == "" ]; do

    # 列出本地地址
    addresses=$(./conflux-toolkit account list)
    addresses=${addresses%Totally*}

    needImport=false
    noAccount="No account found!"
    # 检查是否有账户存在,有则选择账户地址，如果没有用户需要的地址则要求导入
    if [ "$noAccount" != "$addresses" ]; then
        echo "- 当前存在的地址列表"
        echo "$addresses"
        echo "- 请选择用于空投的地址序号, 如果没有您需要的地址请输入 \"N\" 根据提示导入私钥"
        # 选择序号
        while
            true
            read -r -p "" input
        do
            case $input in
            "N")
                needImport=true
                break
                ;;
            *)
                # address=$(echo $addresses | grep "\[${input}\] CFX")
                address=${addresses#*\[${input}\]}
                address="$address["
                if [ "$address" != "" ]; then
                    # address=CFX${address#*CFX}
                    address=${address%%[*}
                    address=$(echo $address | xargs)
                    # echo "您选择了地址: ${address}"
                    needImport=false
                    break
                else
                    echo "输入无效，请重新输入"
                fi
                # echo "输入无效，请重新输入"
                ;;
            esac
        done
    # 没有则要求用户导入
    else
        needImport=true
    fi

    if [ "$needImport" = true ]; then
        echo "- 没有找到账户，请输入您的私钥导入账户"
        read -r -p "" input
        # echo "- 收到私钥：${input}"
        ./conflux-toolkit account import --key ${input}
        # address=$(./conflux-toolkit account import --key ${input})
    fi
done
# 如果不存在导入私钥
# 显示导入的地址并继续

# 不存在则导入

# privateKey=$(<./1_填写发送地址.txt)
# echo "- 发现私钥 : ${privateKey}"
# echo "- 正在导入私钥，将会自动设置密码，设置默认密码为 123"
# ./conflux-toolkit account import --key ${privateKey} <<EOF
# 123
# 123
# EOF

# address=$(./conflux-toolkit account list)
# address="NET${address#*NET}"
echo "- 将使用该地址空投：${address}"

# select network type, mainnet or testnet?
echo "- 请手动输入您要空投到的网络类型：测试网输入test, tethys主网输入tethys"
echo "注意：cToken发送只支持tethys主网"

while
    true
    read -r -p "" input
do
    case $input in
    "test")
        echo "- 您选择的是测试网"
        # url="ws://test.confluxrpc.com"
        url="https://test.confluxrpc.com"
        break
        ;;

    "tethys")
        echo "- 您选择的是Tethys主网"
        # url="ws://main.confluxrpc.com/ws"
        url="https://main.confluxrpc.com"
        break
        ;;

    *)
        echo "输入无效，请重新输入"
        ;;
    esac
done

echo "- 默认 gasPrice 是 1K drip，当网络拥堵时需要调高gasPrice，输入 \"N\" 跳过设置；输入 \"Y\" 提高gasPrice至 1G drip "
read -r -p "" input
while
    true
    read -r -p "" input
do
    case $input in
    "N")
        gasPrice=1000
        break
        ;;
    *) ;;

    "Y")
        gasPrice=1000000000
        break
        ;;
    esac
done

# start airdrop
echo "- 将根据空投列表文件 \"1_填写空投列表.csv\" 开始空投"
echo "- 请根据提示输入账户密码继续\n"
./conflux-toolkit transfer --receivers "./1_填写空投列表.csv" --from ${address} --price ${gasPrice} --weight 1 --url ${url} --batch 500
#  <<EOF
# 123
# EOF

# rm -rf ./keystore
echo "\n- 完成"
