module github.com/GoogleCloudPlatform/compute-image-tools/gce_image_import_export_tests

go 1.13

require (
	cloud.google.com/go v0.53.0 // indirect
	cloud.google.com/go/storage v1.5.0
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0-20200214222452-1f73b9cf8968
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200214222452-1f73b9cf8968
	github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils v0.0.0-20200214222452-1f73b9cf8968
	golang.org/x/exp v0.0.0-20200213203834-85f925bdd4d0 // indirect
	golang.org/x/tools v0.0.0-20200214201135-548b770e2dfa // indirect
	google.golang.org/api v0.17.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../cli_tools
