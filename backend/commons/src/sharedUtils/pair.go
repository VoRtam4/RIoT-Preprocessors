/**
 * @file pair.go
 * @brief Jednoduchý sdílený typ dvojice hodnot.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

type Pair[T any, U any] struct {
	e1 T
	e2 U
}

func NewPairOf[T any, U any](e1 T, e2 U) Pair[T, U] {
	return Pair[T, U]{
		e1: e1,
		e2: e2,
	}
}

func (p Pair[T, U]) GetFirst() T {
	return p.e1
}

func (p Pair[T, U]) GetSecond() U {
	return p.e2
}
