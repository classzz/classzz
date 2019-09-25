package cross

import (
	// "bytes"
	// "encoding/binary"
	// "errors"
	// "math/big"
	"fmt"

	// "github.com/classzz/classzz/chaincfg"
	// "github.com/classzz/classzz/czzec"
	// "github.com/classzz/classzz/txscript"
	// "github.com/classzz/classzz/wire"
	// "github.com/classzz/czzutil"
	"testing"
)


func TestCalcModel(t *testing.T){
	sum := 10
	reserve := []int64{20,100}
	doge := int64(10000)
	itc := int64(100)
	for i:=0;i<sum;i++ {
		v1 := toDoge(reserve[0],doge)
		v2 := toLtc(reserve[1],itc)

		fmt.Println("entangle index:",i,"[doge:",doge," to czz:",v1,"][itc:",itc," to czz:",v2,"]")
		reserve[0] += doge
		reserve[1] += itc
		doge = doge * int64(i+1)
		itc = itc * int64(i+1)
	}
}
