package err4

func Then(err *error, ok func(), catch func()) {
	switch {
	case *err == nil && ok != nil:
		ok()
	case *err != nil && catch != nil:
		catch()
	}
}
