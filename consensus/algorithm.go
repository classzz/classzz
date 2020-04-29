package consensus

import (
	"archive/zip"
	"bytes"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io"
)

const TBLSize = 8388608

type CZZTBL struct {
	data  []uint64 // The actual cache data content
	bflag int
}

var czzTbl *CZZTBL

func init() {
	czzTbl = &CZZTBL{
		data:  make([]uint64, TBLSize/8),
		bflag: 0,
	}
}

func shift2048(in []uint64, sf int) int {
	var sfI int = sf / 64
	var sfR int = sf % 64
	var mask uint64 = (uint64(1) << uint(sfR)) - 1
	var bits int = (64 - sfR)
	var res uint64
	if sfI == 1 {
		val := in[0]
		for k := 0; k < 31; k++ {
			in[k] = in[k+1]
		}
		in[31] = val
	}
	res = (in[0] & mask) << uint(bits)
	for k := 0; k < 31; k++ {
		var val uint64 = (in[k+1] & mask) << uint(bits)
		in[k] = (in[k] >> uint(sfR)) + val
	}
	in[31] = (in[31] >> uint(sfR)) + res
	return 0
}

func xor64(val uint64) int {
	var r int = 0
	for k := 0; k < 64; k++ {
		r ^= int(val & 0x1)
		val = val >> 1
	}
	return r
}
func muliple(input []uint64, prow []uint64) uint {
	var r int = 0
	for k := 0; k < 32; k++ {
		if input[k] != 0 && prow[k] != 0 {
			r ^= xor64(input[k] & prow[k])
		}
	}

	return uint(r)
}

func MatMuliple(input []uint64, output []uint64, pmat []uint64) int {
	prow := pmat[:]
	var point uint = 0

	for k := 0; k < 2048; k++ {
		kI := k / 64
		kR := k % 64
		var temp uint
		temp = muliple(input[:], prow[point:])

		output[kI] |= (uint64(temp) << uint(kR))
		point += 32
	}

	return 0
}

func hashCZZA(b []byte) []byte {
	hash := sha3.Sum512(b)
	for i := 0; i < 32; i++ {
		hash[i], hash[63-i] = hash[63-i], hash[i]
	}
	first := append(hash[:], hash[:]...)

	return append(first, first...)
}

func hashCZZB(data_in []uint64, plookup []uint64) int {
	var ptbl []uint64
	var data_out [32]uint64
	for k := 0; k < 64; k++ {

		sf := int(data_in[0] & 0x7f)
		bs := int(data_in[31] >> 60)

		ptbl = plookup[bs*2048*32:]

		MatMuliple(data_in[:], data_out[:], ptbl[:])
		shift2048(data_out[:], sf)

		for k := 0; k < 32; k++ {
			data_in[k] = data_out[k]
			data_out[k] = 0
		}
	}
	return 0
}

func hashCZZC(b []byte) []byte {
	hash := sha3.Sum256(b)

	return hash[:]
}

func HashCZZ(header []byte, nonce uint64) []byte {
	var seed [64]byte

	err := getTBLFromZip()
	if err != 0 {
		panic("Init Table failed")
	}
	val0 := uint32(nonce & 0xFFFFFFFF)
	val1 := uint32(nonce >> 32)

	for k := 3; k >= 0; k-- {
		seed[k] = byte(val0) & 0xFF
		val0 >>= 8
	}
	for k := 7; k >= 4; k-- {
		seed[k] = byte(val1) & 0xFF
		val1 >>= 8
	}

	for k := 0; k < 32; k++ {
		seed[k+8] = header[k]
	}
	first := hashCZZA(seed[:])
	var data [32]uint64

	for k := 0; k < 8; k++ {
		for x := 0; x < 8; x++ {
			var sft int = x * 8
			val := (uint64(first[k*8+x]) << uint(sft))
			data[k] += val
		}
	}
	for k := 1; k < 4; k++ {
		for x := 0; x < 8; x++ {
			data[k*8+x] = data[x]
		}
	}

	hashCZZB(data[:], czzTbl.data[:])

	var dat_in [256]byte
	for k := 0; k < 32; k++ {
		val := data[k]
		for x := 0; x < 8; x++ {
			dat_in[k*8+x] = byte(val & 0xFF)
			val = val >> 8
		}
	}

	for k := 0; k < 64; k++ {
		var temp byte
		temp = dat_in[k*4]
		dat_in[k*4] = dat_in[k*4+3]
		dat_in[k*4+3] = temp
		temp = dat_in[k*4+1]
		dat_in[k*4+1] = dat_in[k*4+2]
		dat_in[k*4+2] = temp
	}

	result := hashCZZC(dat_in[:])
	return result[:]
}

// generate ensures that the dataset content is generated before use.
func getTBLFromZip() int {
	tbl_standard := [32]byte{211, 78, 111, 5, 122, 176, 245, 7, 58, 142, 149, 100, 165, 70, 120, 84, 78, 57, 234, 75, 247, 160, 26, 46, 22, 71, 62, 160, 194, 110, 123, 159}

	if czzTbl.bflag == 1 {
		return 0
	}
	r, err := zip.OpenReader("csatable.zip")
	if err != nil {
		log.Error(err)
		return 1
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != "csatable.bin" {
			continue
		}

		file, err := f.Open()
		if err != nil {
			fmt.Println(err)
		}
		defer file.Close()
		var size = f.FileInfo().Size()
		if size != TBLSize {
			log.Error("file size wrong!")
			return 1
		}

		buf1 := bytes.NewBuffer([]byte{})
		cb := make([]byte, 1024)
		//file
		for {
			n, err := file.Read(cb)
			if err != nil && err != io.EOF {
				panic(err)
			}
			if 0 == n {
				break
			}
			buf1.Write(cb[:n])
		}

		table_hash := sha3.Sum256(buf1.Bytes())
		if !bytes.Equal(table_hash[:], tbl_standard[:]) {
			log.Error("Read Miner Table Failed ")
			return 1
		}

		k := uint(0)
		for k = 0; k < TBLSize/8; k++ {
			kk := uint(0)
			val_t := uint64(0)
			for kk = 0; kk < 8; kk++ {
				val_t += uint64(buf1.Bytes()[k*8+kk]) << (kk * 8)
			}
			czzTbl.data[k] = val_t
		}
		czzTbl.bflag = 1
		return 0
	}

	return 1
}
