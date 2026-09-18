module github.com/WilsonSayago/initModules/examples/standalone

go 1.24.0

toolchain go1.27.1

require github.com/WilsonSayago/initModules/v2 v2.0.0

require (
	github.com/magiconair/properties v1.18.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/WilsonSayago/initModules/v2 => ../..
