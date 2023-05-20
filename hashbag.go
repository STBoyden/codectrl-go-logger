package codectrl

type hashbag[K comparable] map[K]uint32

func hashbagNew[K comparable]() hashbag[K] {
	return make(map[K]uint32)
}

func hashbagInsert[K comparable](bag hashbag[K], key K) {
	bag[key]++
}

func hashbagRemove[K comparable](bag hashbag[K], key K) {
	if bag[key] > 0 {
		bag[key]--
		return
	}

	delete(bag, key)
}
