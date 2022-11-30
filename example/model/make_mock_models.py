from argparse import ArgumentParser
import tensorflow as tf
from tensorflow.keras import Input, Model
from tensorflow.keras.layers import Concatenate, Lambda
from tensorflow.keras.layers.experimental.preprocessing import StringLookup, TextVectorization

if __name__ == '__main__':
    ap = ArgumentParser(description='Generate TensorFlow 2.4.x models.')
    ap.add_argument('--vocab-size', type=int)

    ap.add_argument('--no-vectorization', action='store_true')
    ap.add_argument('--vectorization-dir', default='vec_model')

    ap.add_argument('--no-string-lookup', action='store_true')
    ap.add_argument('--string-lookup-dir', default='sls_model')

    pa, _ = ap.parse_known_args()

    vocab_data = [chr(ord('a') + i) for i in range(pa.vocab_size)]
    print(vocab_data)

    test_tv_data = ['a d e e', 'b c d', 'a d e a c', 'a b e a']
    test_tv_tensor = tf.constant(test_tv_data)

    test_sl_tensor = tf.constant(['a', 'e', 'd', 'c'])

    # TODO specify test data, generate a subset of models from arguments
    if not pa.no_vectorization:
        tv_i = Input(shape=(1,), name='tv', dtype='string')
        tv_layer = TextVectorization(name='tv_text_vectorization')


        tv_layer.adapt(test_tv_tensor)


        sl_i = Input(shape=(1,), name='sl', dtype='string')
        sl_layer = StringLookup(name='sl_string_lookup')
        sl_layer.adapt(vocab_data)

        o = Concatenate()([tv_layer(tv_i), sl_layer(sl_i)])
        o = Lambda(lambda t: tf.math.reduce_prod(tf.math.add(t, 1), axis=1), name='reduce_sum')(o)

        vec_model = Model(inputs=[tv_i, sl_i], outputs=o, name='test_vectorization')

        print(vec_model.summary())
        print(vec_model.predict({'tv': test_tv_tensor, 'sl': test_sl_tensor}))

        vec_model.save(pa.vectorization_dir)

    if not pa.no_string_lookup:
        # StringLookup only model
        sl_i = Input(shape=(1,), name='sl', dtype='string')
        sl_layer = StringLookup(name='sl_string_lookup')
        sl_layer.adapt(vocab_data)

        sa_i = Input(shape=(1,), name='sa', dtype='string')
        sa_layer = StringLookup(name='sa_string_lookup')
        sa_layer.adapt(vocab_data)

        o = Concatenate()([sa_layer(sa_i), sl_layer(sl_i)])
        o = Lambda(lambda t: tf.math.reduce_prod(t, axis=1), name='reduce_sum')(o)

        sls_model = Model(inputs=[sa_i, sl_i], outputs=o, name='test_string_lookups')
        print(sls_model.summary())
        print(sls_model.predict({'sa': test_tv_tensor, 'sl': test_sl_tensor}))

        sls_model.save(pa.string_lookup_dir)

