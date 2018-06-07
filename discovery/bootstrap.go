package discovery

func Bootstrap() error {
	gdp, err := ListGoDocPackages()
	if err != nil {
		return err
	}
	return nil
}
