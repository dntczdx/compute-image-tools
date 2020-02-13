module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.13

require (
	cloud.google.com/go v0.52.0
	cloud.google.com/go/storage v1.5.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200211213215-e75aedeb435d
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200206014411-426b6301f679
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20200211005319-080372593330
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.4
	github.com/golang/mock v1.3.1
	github.com/google/logger v1.0.1
	github.com/google/uuid v1.1.1
	github.com/klauspost/compress v1.10.0 // indirect
	github.com/klauspost/pgzip v1.2.1
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.0
	github.com/stretchr/testify v1.4.0
	github.com/vmware/govmomi v0.22.1
	golang.org/x/exp v0.0.0-20200207192155-f17229e696bd // indirect
	golang.org/x/mod v0.2.0 // indirect
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2 // indirecresource_labeler_testt
	golang.org/x/sys v0.0.0-20200202164722-d101bd2416d5
	golang.org/x/tools v0.0.0-20200211205636-11eff242d136 // indirect
	google.golang.org/api v0.17.0
	google.golang.org/genproto v0.0.0-20200211111953-2dc5924e3898 // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
