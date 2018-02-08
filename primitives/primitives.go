package primitives

import (
	"crypto/sha256"
)

type AuthSet struct {
	IdentityList []Identity
	StatusArray  []int
	IdentityMap  map[Identity]int
}

func (a *AuthSet) New() {
	a.IdentityList = make([]Identity, 0)
	a.StatusArray = make([]int, 0)
	a.IdentityMap = make(map[Identity]int)
}
func (a *AuthSet) Add(id Identity, status int) int {
	index := len(a.IdentityList)
	a.IdentityMap[id] = index
	a.IdentityList[index] = id
	a.StatusArray[index] = status
}
func (a *AuthSet) IsLeader(id Identity) bool{
	index, ok := a.IdentityMap[id]
	if(!ok){panic("Bad Identity")}
	return a.StatusArray[index]>=0
}


type Identity int

type Hash [sha256.Size]byte
