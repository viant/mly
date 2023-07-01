# Example server configuration

This specific configuration is tailored to have replacements configured by [`endly`](https://github.com/viant/endly).

To manually use this, replace all instances of `${appPath}` with the path to the root of the repository.

Additionally, this configuration is expecting an Aerospike instance running on port 3000. 
To run without Aerospike, remove the `connections` object, as well as each `connection` attribute in the `datastores` array.

