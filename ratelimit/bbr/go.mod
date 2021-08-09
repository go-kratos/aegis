module github.com/go-kratos/sra/ratelimit/bbr

go 1.16

require (
	github.com/go-kratos/sra/pkg/cpu v0.0.0-00010101000000-000000000000
	github.com/go-kratos/sra/pkg/window v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
)

replace github.com/go-kratos/sra/pkg/window => ../../pkg/window

replace github.com/go-kratos/sra/pkg/cpu => ../../pkg/cpu
