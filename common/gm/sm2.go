/*
Copyright Suzhou Tongji Fintech Research Institute 2017 All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gm

// reference to ecdsa
import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/asn1"
	"io"
	"math/big"
	"sync"

	"github.com/rongzer/blockchain/common/log"
)

var (
	P, _  = new(big.Int).SetString("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000FFFFFFFFFFFFFFFF", 16)
	A, _  = new(big.Int).SetString("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000FFFFFFFFFFFFFFFC", 16)
	B, _  = new(big.Int).SetString("28E9FA9E9D9F5E344D5A9E4BCF6509A7F39789F515AB8F92DDBCBD414D940E93", 16)
	N, _  = new(big.Int).SetString("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFF7203DF6B21C6052B53BBF40939D54123", 16)
	Gx, _ = new(big.Int).SetString("32C4AE2C1F1981195F9904466A39C9948FE30BBFF2660BE1715A4589334C74C7", 16)
	Gy, _ = new(big.Int).SetString("BC3736A2F4F6779C59BDCEE36B692153D0A9877CC62A474002DF32E52139F0A0", 16)
	one   = new(big.Int).SetInt64(1)
)

var initonce sync.Once
var p256Sm2Params *elliptic.CurveParams

func initP256Sm2() {
	p256Sm2Params = &elliptic.CurveParams{Name: "SM2-P-256"} // 注明为SM2
	//SM2椭	椭 圆 曲 线 公 钥 密 码 算 法 推 荐 曲 线 参 数
	p256Sm2Params.P = P
	p256Sm2Params.B = B
	p256Sm2Params.N = N
	p256Sm2Params.Gx = Gx
	p256Sm2Params.Gy = Gy
	p256Sm2Params.BitSize = 256
}

func P256Sm2() elliptic.Curve {
	initonce.Do(initP256Sm2)
	return p256Sm2Params
}

func randFieldElement(c elliptic.Curve, rand io.Reader) (k *big.Int, err error) {
	params := c.Params()
	b := make([]byte, params.BitSize/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}
	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}

func sm2Sign(md []byte, priv *ecdsa.PrivateKey) (r, s *big.Int, err error) {
	c := P256Sm2()

	var k *big.Int
	e := new(big.Int).SetBytes(md)
	rd := rand.Reader
	for { // 调整算法细节以实现SM2
		for {
			k, err = randFieldElement(c, rd)

			if err != nil {
				r = nil
				return
			}
			r, _ = priv.Curve.ScalarBaseMult(k.Bytes())
			r.Add(r, e)
			r.Mod(r, N)
			if r.Sign() != 0 {
				break
			}
			if t := new(big.Int).Add(r, k); t.Cmp(N) == 0 {
				break
			}
		}
		rD := new(big.Int).Mul(priv.D, r)
		s = new(big.Int).Sub(k, rD)
		d1 := new(big.Int).Add(priv.D, one)
		d1Inv := new(big.Int).ModInverse(d1, N)
		s.Mul(s, d1Inv)
		s.Mod(s, N)
		if s.Sign() != 0 {
			break
		}
	}
	return
}

func sm2Verify(hash []byte, pub *ecdsa.PublicKey, r, s *big.Int) bool {
	c := P256Sm2()

	if r.Sign() <= 0 || s.Sign() <= 0 {
		return false
	}
	if r.Cmp(N) >= 0 || s.Cmp(N) >= 0 {
		return false
	}

	// 调整算法细节以实现SM2
	t := new(big.Int).Add(r, s)
	t.Mod(t, N)
	if N.Sign() == 0 {
		return false
	}
	var x *big.Int
	x1, y1 := c.ScalarBaseMult(s.Bytes())
	x2, y2 := c.ScalarMult(pub.X, pub.Y, t.Bytes())
	x, _ = c.Add(x1, y1, x2, y2)
	e := new(big.Int).SetBytes(hash)
	x.Add(x, e)
	x.Mod(x, N)
	return x.Cmp(r) == 0
}

func sm2GetZ(userId []byte, pub *ecdsa.PublicKey) []byte {
	hash := Sm3Hash()
	len := len(userId) * 8

	b := make([]byte, 1)
	b[0] = byte(len >> 8 & 0xff)
	hash.Write(b)
	b1 := make([]byte, 1)
	b1[0] = byte(len & 0xff)
	hash.Write(b1)
	hash.Write(userId)
	A, _ := new(big.Int).SetString("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000FFFFFFFFFFFFFFFC", 16)
	hash.Write(byteConvert32Bytes(A))
	hash.Write(byteConvert32Bytes(B))
	hash.Write(byteConvert32Bytes(Gx))
	hash.Write(byteConvert32Bytes(Gy))
	hash.Write(byteConvert32Bytes(pub.X))
	hash.Write(byteConvert32Bytes(pub.Y))

	return hash.Sum(nil)[:32]
}

func byteConvert32Bytes(n *big.Int) []byte {
	br := n.Bytes()
	if len(br) == 33 {
		return br[1:]
	}
	if len(br) == 32 {
		return br
	}

	if len(br) < 32 {
		tmp := make([]byte, 32)
		for i := 0; i < 32; i++ {
			tmp[0] = byte('0')
			if i >= 32-len(br) {
				tmp[i] = br[i-(32-len(br))]
			}
		}
		return tmp
	}
	return br
}

func Sm2Sign(userId, msg []byte, priv *ecdsa.PrivateKey) (signature []byte, r, s *big.Int, err error) {
	hash := Sm3Hash()
	pub := priv.PublicKey

	z := sm2GetZ(userId, &pub)
	hash.Write(z)
	hash.Write(msg)
	md := hash.Sum(nil)[:32]

	r, s, err = sm2Sign(md, priv)

	signature, err = asn1.Marshal(SM2Signature{R: r, S: s, UserId: userId})
	return
}

type SM2Signature struct {
	R, S   *big.Int
	UserId []byte `asn1:"optional,explicit,tag:0"`
}

func Sm2Verify(userId, msg []byte, pub *ecdsa.PublicKey, signature []byte) bool {

	sig := new(SM2Signature)

	_, err := asn1.Unmarshal(signature, sig)
	if err != nil {
		log.Logger.Errorf("\nSm2Verify Unmarshal signature err:%s\n", err)
		return false
	}

	//如果未传入userId,则尝试从signature中解释
	if len(userId) < 1 {
		userId = sig.UserId
	}

	hash := Sm3Hash()

	z := sm2GetZ(userId, pub)
	hash.Write(z)
	hash.Write(msg)
	md := hash.Sum(nil)[:32]

	return sm2Verify(md, pub, sig.R, sig.S)
}
