package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

type AuditData struct {
	ComX  *big.Int
	ComY  *big.Int
	ComQX *big.Int
	ComQY *big.Int
	Tx1   *big.Int
	Ty1   *big.Int
	Tx2   *big.Int
	Ty2   *big.Int
	Z     *big.Int
	Zr1   *big.Int
	Zr2   *big.Int
}

func main() {
	fmt.Println("正在读取相应承诺和NIZK证明······")

	filePath := filepath.Join(".", "comments.txt")
	data, err := readAuditData(filePath)
	if err != nil {
		fmt.Printf("读取或解析 comments.txt 失败: %v\n", err)
		os.Exit(1)
	}

	passed := verifyNIZKWithProgress(data)
	if passed {
		fmt.Println("验证通过")
	} else {
		fmt.Println("验证未通过")
	}
}

func readAuditData(filePath string) (*AuditData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开 %s: %w", filePath, err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("第 %d 行格式错误，应为 key: value", lineNo)
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if value == "" {
			return nil, fmt.Errorf("第 %d 行的 %s 值为空", lineNo, key)
		}
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	get := func(key string) (*big.Int, error) {
		value, ok := values[strings.ToLower(key)]
		if !ok {
			return nil, fmt.Errorf("缺少字段 %s", key)
		}
		n, err := parseBigInt(value)
		if err != nil {
			return nil, fmt.Errorf("字段 %s 解析失败: %w", key, err)
		}
		return n, nil
	}

	var err error
	data := &AuditData{}
	if data.ComX, err = get("comx"); err != nil {
		return nil, err
	}
	if data.ComY, err = get("comy"); err != nil {
		return nil, err
	}
	if data.ComQX, err = get("comqx"); err != nil {
		return nil, err
	}
	if data.ComQY, err = get("comqy"); err != nil {
		return nil, err
	}
	if data.Tx1, err = get("tx1"); err != nil {
		return nil, err
	}
	if data.Ty1, err = get("ty1"); err != nil {
		return nil, err
	}
	if data.Tx2, err = get("tx2"); err != nil {
		return nil, err
	}
	if data.Ty2, err = get("ty2"); err != nil {
		return nil, err
	}
	if data.Z, err = get("z"); err != nil {
		return nil, err
	}
	if data.Zr1, err = get("zr1"); err != nil {
		return nil, err
	}
	if data.Zr2, err = get("zr2"); err != nil {
		return nil, err
	}

	return data, nil
}

func parseBigInt(raw string) (*big.Int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("空数字")
	}

	base := 10
	if strings.HasPrefix(strings.ToLower(s), "0x") {
		base = 16
		s = s[2:]
	} else if containsHexLetter(s) {
		base = 16
	}

	n := new(big.Int)
	if _, ok := n.SetString(s, base); !ok {
		return nil, fmt.Errorf("非法数字 %q", raw)
	}
	if n.Sign() < 0 {
		return nil, fmt.Errorf("不能为负数 %q", raw)
	}
	return n, nil
}

func containsHexLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			return true
		}
	}
	return false
}

func verifyNIZKWithProgress(data *AuditData) bool {
	showProgress(0)
	pause()

	curve := secp256k1.S256()
	showProgress(15)
	pause()

	if !curve.IsOnCurve(data.ComX, data.ComY) ||
		!curve.IsOnCurve(data.ComQX, data.ComQY) ||
		!curve.IsOnCurve(data.Tx1, data.Ty1) ||
		!curve.IsOnCurve(data.Tx2, data.Ty2) {
		showProgress(100)
		return false
	}
	showProgress(30)
	pause()

	challengeInput := data.Tx1.String() + data.Ty1.String() + data.Tx2.String() + data.Ty2.String()
	challenge := sha256.Sum256([]byte(challengeInput))
	showProgress(45)
	pause()

	// 这里的 H 点沿用原代码中的固定点坐标。
	// 该坐标实际也是 secp256k1 标准生成元 G 的坐标；这里保持原证明逻辑不改变。
	HX, _ := new(big.Int).SetString("C6047F9441ED7D6D3045406E95C07CD85C778E4B8CEF3CA7ABAC09B95C709EE5", 16)
	HY, _ := new(big.Int).SetString("1AE168FEA63DC339A3C58419466EFAEEF7F632653266D0E1236431A950CFE52A", 16)
	if !curve.IsOnCurve(HX, HY) {
		showProgress(100)
		return false
	}
	showProgress(60)
	pause()

	gzx, gzy := curve.ScalarBaseMult(data.Z.Bytes())
	hzrx1, hzry1 := curve.ScalarMult(HX, HY, data.Zr1.Bytes())
	hzrx2, hzry2 := curve.ScalarMult(HX, HY, data.Zr2.Bytes())
	comInCX, comInCY := curve.ScalarMult(data.ComX, data.ComY, challenge[:])
	comOutCX, comOutCY := curve.ScalarMult(data.ComQX, data.ComQY, challenge[:])
	if anyNil(gzx, gzy, hzrx1, hzry1, hzrx2, hzry2, comInCX, comInCY, comOutCX, comOutCY) {
		showProgress(100)
		return false
	}
	showProgress(75)
	pause()

	leftX1, leftY1 := curve.Add(gzx, gzy, hzrx1, hzry1)
	leftX2, leftY2 := curve.Add(gzx, gzy, hzrx2, hzry2)
	rightX1, rightY1 := curve.Add(data.Tx1, data.Ty1, comInCX, comInCY)
	rightX2, rightY2 := curve.Add(data.Tx2, data.Ty2, comOutCX, comOutCY)
	if anyNil(leftX1, leftY1, leftX2, leftY2, rightX1, rightY1, rightX2, rightY2) {
		showProgress(100)
		return false
	}
	showProgress(90)
	pause()

	flag1 := leftX1.Cmp(rightX1) == 0 && leftY1.Cmp(rightY1) == 0
	flag2 := leftX2.Cmp(rightX2) == 0 && leftY2.Cmp(rightY2) == 0

	showProgress(100)
	return flag1 && flag2
}

func anyNil(values ...*big.Int) bool {
	for _, value := range values {
		if value == nil {
			return true
		}
	}
	return false
}

func showProgress(percent int) {
	const width = 30
	filled := percent * width / 100
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", width-filled)
	fmt.Printf("\r验证进度: [%s] %3d%%", bar, percent)
	if percent == 100 {
		fmt.Println()
	}
}

func pause() {
	time.Sleep(120 * time.Millisecond)
}
