from argparse import ArgumentParser
import itertools
import random
import tensorflow as tf
from tensorflow.keras import Input, Model
from tensorflow.keras.layers import Concatenate, Lambda
from tensorflow.keras.layers.experimental.preprocessing import StringLookup, TextVectorization

if __name__ == '__main__':
    ap = ArgumentParser(description='Generate TensorFlow 2.4.x models.')

    # incomplete, docs provide a lot more entropy
    ap.add_argument('--seed', type=int)

    ap.add_argument('--vocab-size', type=int, default=3)
    ap.add_argument('--max-test-vector-length', type=int, default=8)
    ap.add_argument('--num-test-samples', type=int, default=4)

    ap.add_argument('--no-vectorization', action='store_true')
    ap.add_argument('--no-string-lookup', action='store_true')

    ap.add_argument('--no-int-out', action='store_true')
    ap.add_argument('--no-float-out', action='store_true')

    pa, _ = ap.parse_known_args()

    if pa.seed:
        random.seed(pa.seed)

    vocab_data = [chr(ord('a') + i) for i in range(pa.vocab_size)]
    print(vocab_data)

    oov_element = chr(ord('a') + pa.vocab_size)
    vocab_with_oov = vocab_data + [oov_element]

    test_sl_data = []
    test_tv_data = []

    for ti in range(pa.num_test_samples):
        test_sl_data.append(random.choice(vocab_data))

        vec_len = random.randint(1, pa.max_test_vector_length)
        test_tv_row = random.choices(vocab_with_oov, k=vec_len)
        test_tv_data.append(' '.join(test_tv_row))

    test_sl_data = test_sl_data[:-1]
    test_sl_data.append(oov_element)

    print(test_sl_data)
    print(test_tv_data)

    test_sl_tensor = tf.constant(test_sl_data)
    test_tv_tensor = tf.constant(test_tv_data)

    def inputs_with_vec():
        tv_i = Input(shape=(1,), name='tv', dtype='string')
        tv_layer = TextVectorization(name='tv_text_vectorization')
        tv_layer.adapt(test_tv_tensor)

        sl_i = Input(shape=(1,), name='sl', dtype='string')
        sl_layer = StringLookup(name='sl_string_lookup')
        sl_layer.adapt(vocab_data)

        o = Concatenate()([tv_layer(tv_i), sl_layer(sl_i)])

        return [tv_i, sl_i], o

    def inputs_sl_only():
        # StringLookup only model
        sl_i = Input(shape=(1,), name='sl', dtype='string')
        sl_layer = StringLookup(name='sl_string_lookup')
        sl_layer.adapt(vocab_data)

        sa_i = Input(shape=(1,), name='sa', dtype='string')
        sa_layer = StringLookup(name='sa_string_lookup')
        sa_layer.adapt(vocab_data)

        o = Concatenate()([sa_layer(sa_i), sl_layer(sl_i)])
        return [sl_i, sa_i], o

    input_generators = [(
            pa.no_vectorization, 
            'vectorization', 
            inputs_with_vec, 
            {'tv': test_tv_tensor, 'sl': test_sl_tensor}
        ), 
        (
            pa.no_string_lookup, 
            'string_lookups', 
            inputs_sl_only, 
            {'sa': test_tv_tensor, 'sl': test_sl_tensor}
        )]

    output_mods = [(pa.no_float_out, True, 'float'), 
                   (pa.no_int_out, False, 'int')]

    for gen_params, mod_params in itertools.product(input_generators, output_mods):
        is_disabled, gen_name, generator, tester = gen_params
        if is_disabled:
            continue

        # note variable name overlap
        is_disabled, include_divide, mod_name = mod_params
        if is_disabled:
            continue

        inputs, o = generator()

        o = Lambda(lambda t: tf.math.add(t, 1), name='add_one')(o)
        o = Lambda(lambda t: tf.math.reduce_prod(t, axis=1), name='reduce_prod')(o)

        if include_divide:
            o = Lambda(lambda t: tf.math.divide(t, 7), name='divide')(o)

        o = Lambda(lambda t: tf.expand_dims(t, axis=1), name='expand')(o)

        check_model = Model(inputs=inputs, outputs=o, name=f'{gen_name}_{mod_name}')

        print({i.name: i.dtype for i in check_model.inputs})
        print({o.name: o.dtype for o in check_model.outputs})

        print(check_model.summary())
        print(check_model.predict(tester))

        save_dir = f'{gen_name}_{mod_name}_model'
        print(f'Saving model to {save_dir}...')
        check_model.save(save_dir)
        print(f'Saved model to {save_dir}.')

    print('Done.')

