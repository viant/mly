# Example and test models

Use Python with TensorFlow to generate these models.
The `SavedModel` format has shown no issues with backwards compatibility until v2.12.0+.

Requires Python (3.7 max) with Tensorflow 2.4.3 max.
Use image `tensorflow/tensorflow:2.4.3` for easiest setup.

I.e. `docker run --rm -it -v "$PWD":/w -w /w tensorflow/tensorflow:2.4.3 python make_mock_models.py`

## Some cases

### Input and caching

1. Input String lookup layer - for testing input caching given a fixed input space.
2. Input Text vectorization layer - for testing input caching given a fixed, differently shaped input space (unimplemented).

### Output and transforming

1. Integer - OK.
2. Float - OK.
3. Dictionary of batches - OK.
4. Batch of dictionaries - unsupported, untested.
