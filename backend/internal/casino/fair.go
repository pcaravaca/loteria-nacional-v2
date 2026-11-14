package casino

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Genera la semilla del servidor y su hash. El hash se publica antes de jugar
// para comprometer el resultado; la semilla se revela al verificar.
func NuevaSemilla() (semilla, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	semilla = hex.EncodeToString(buf)
	return semilla, HashSemilla(semilla), nil
}

func HashSemilla(semilla string) string {
	sum := sha256.Sum256([]byte(semilla))
	return hex.EncodeToString(sum[:])
}

// Sortea `cuantos` valores en [0, max) de forma determinista a partir de la
// semilla del servidor, la semilla del cliente y el nonce de la ronda.
// Cualquiera con la semilla revelada puede reproducir el mismo resultado.
func Sortear(semillaServidor, semillaCliente string, nonce int64, cuantos, max int) []int {
	if max <= 0 {
		max = 1
	}
	out := make([]int, cuantos)
	for i := 0; i < cuantos; i++ {
		mac := hmac.New(sha256.New, []byte(semillaServidor))
		fmt.Fprintf(mac, "%s:%d:%d", semillaCliente, nonce, i)
		sum := mac.Sum(nil)
		v := binary.BigEndian.Uint32(sum[:4])
		out[i] = int(v % uint32(max))
	}
	return out
}
