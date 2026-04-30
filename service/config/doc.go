package config

/*
Package service/config defines ingestible (and currently mixed reporting of) configuration values for mly.

# General Usage Order

1. Instantiate the config object.
2. Set fields as needed.
3. Call Init() to normalize and set defaults.
4. Call Validate() to check base field validation. Calling Validate() before Init() may catch false positive validation errors.
5. Call ConfigCheck() to check relationship validation.

# Validation

These are the following types of validation:

## Base field validation

These will check to make sure immediate fields are valid.
This is done via a method named Validate().
This will check Storables and Transformers as well.

## Relationship validation

These will check to make sure relationships with other config entities are valid.
This is separate from Validate() for backwards compatibility.
Eventually, the signatures for Validate() will be updated to include these checks.
These checks can be done via an argument flag, as well as on startup.
This is done via a method named ConfigCheck().

## External validation

These checks occur on startup.

These checks include:

1. Checking the model configured inputs and outputs match the actual inputs and outputs as defined by a model.
	Failures here can occur if the model names, types, or shapes cannot be resolved to match the configured inputs and outputs.
	These checks cannot be done independently of loading actual objects.
2. Checking that Triton servers can be connected to.
3. Checking that Aerospike servers can be connected to.

*/
