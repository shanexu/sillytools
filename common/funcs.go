package common

type StrErrFunc = func() (string, error)

func Alternatives(funs ...StrErrFunc) StrErrFunc {
	return func() (string, error) {
		var s string
		var err error
		for _, f := range funs {
			s, err = f()
			if err == nil {
				return s, nil
			}
		}
		return s, err
	}
}
