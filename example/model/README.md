# Example and test models

Use Python with TensorFlow to generate these models.
The `SavedModel` format has shown no issues with backwards compatibility, so any version should be safe.

For the purposes of testing, we test the string lookup, int output, vectorization, and float outputs; this means we just need two models, one for int and another for float lookups. 
Both models have a string lookup layer.
