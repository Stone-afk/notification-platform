package helloworldv1

func (x *SayHelloRequest) GetIdempotencyKeys() []string {
	if x != nil {
		return x.GetKeys()
	}
	return nil
}
