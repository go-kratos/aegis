module github.com/go-kratos/sra/ratelimit/bbr

go 1.16

require (
	github.com/go-kratos/sra/pkg/cpu v0.0.0-20210809055913-feb6d74203a8
	github.com/go-kratos/sra/pkg/window v0.0.0-20210809055913-feb6d74203a8
	github.com/stretchr/testify v1.7.0
)

replace (
	github.com/go-kratos/sra/pkg/cpu => ../../pkg/cpu
	github.com/go-kratos/sra/pkg/window => ../../pkg/window
)