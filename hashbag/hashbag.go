package hashbag

type HashBag[K comparable] map[K]uint32

func New[K comparable]() HashBag[K] {
	return make(map[K]uint32)
}

func Insert[K comparable](bag HashBag[K], key K) {
	bag[key]++
}

func Remove[K comparable](bag HashBag[K], key K) {
	if bag[key] > 0 {
		bag[key]--
		return
	}

	delete(bag, key)
}
