package diagnostics

func (s *Service) GetDefaultGateway() (*GatewayInfo, error) {
	return getDefaultGateway()
}
