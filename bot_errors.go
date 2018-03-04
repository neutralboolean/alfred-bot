type RSVPError struct {
	msg string
}

func (re *RSVPError) Error() string { return re.msg }