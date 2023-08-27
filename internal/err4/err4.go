package err4

func Then(err *error, ok func(), catch func()) {
	if *err == nil {
		if ok != nil {
			ok()
		}
	} else {
		if catch != nil {
			catch()
		}
	}
}
