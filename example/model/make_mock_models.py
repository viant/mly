from argparse import ArgumentParser
import itertools
import numpy as np
import random
import tensorflow as tf
import time
from tensorflow.keras import Input, Model
from tensorflow.keras.layers import Concatenate, Lambda, Layer
from tensorflow.keras.layers.experimental.preprocessing import StringLookup, TextVectorization

class LookupLayer(Layer):
    def __init__(self, keys, values, default_value, name=None, table_name=None):
        self.keys = keys 
        self.values = values
        self.default_value = default_value
        self.table_name = table_name
        super().__init__(name=name, trainable=False)

    def get_config(self):
        return dict()

    def build(self, input_shape):
        keys_tensor = tf.constant(self.keys)
        values_tensor = tf.constant(self.values)
        kv_init = tf.lookup.KeyValueTensorInitializer(keys=keys_tensor, values=values_tensor)
        table = tf.lookup.StaticHashTable(kv_init, default_value=self.default_value, name=self.table_name)
        self.table = table

    def call(self, inputs):
        return self.table.lookup(inputs)

if __name__ == '__main__':
    ap = ArgumentParser(description='Generate TensorFlow 2.4.x models.')

    # incomplete, docs provide a lot more entropy
    ap.add_argument('--seed', type=int)

    ap.add_argument('--vocab-size', type=int, default=3)
    ap.add_argument('--max-test-vector-length', type=int, default=8)
    ap.add_argument('--num-test-samples', type=int, default=4)

    ap.add_argument('--no-vectorization', action='store_true')
    ap.add_argument('--no-string-lookup', action='store_true')
    ap.add_argument('--no-float-input', action='store_true')

    ap.add_argument('--no-int-out', action='store_true')
    ap.add_argument('--no-float-out', action='store_true')

    ap.add_argument('--no-lookup-layer', action='store_true', help='for broken_case_v0_5_0 model')
    ap.add_argument('--no-keyed-out', action='store_true', help='for testing v0.5.0')

    ap.add_argument('--no-slow', action='store_true', help='for testing thread limit')
    ap.add_argument('--slow-repeat', type=int, default=10, help='looping math operations for slow model')

    pa, _ = ap.parse_known_args()

    if pa.seed:
        random.seed(pa.seed)

    vocab_size = pa.vocab_size
    vocab_data = [chr(ord('a') + i) for i in range(vocab_size)]
    print(vocab_data)

    oov_element = chr(ord('a') + vocab_size)
    vocab_with_oov = vocab_data + [oov_element]

    test_sl_data = []
    test_tv_data = []
    test_ll_data = []
    test_fl_data = []

    for ti in range(pa.num_test_samples):
        test_sl_data.append(random.choice(vocab_data))

        vec_len = random.randint(1, pa.max_test_vector_length)
        test_tv_row = random.choices(vocab_with_oov, k=vec_len)
        test_tv_data.append(' '.join(test_tv_row))

        test_ll_data.append(random.randint(0, vocab_size - 1))

        test_fl_data.append(random.random())

    test_sl_data = test_sl_data[:-1]
    test_sl_data.append(oov_element)

    print(test_sl_data)
    print(test_tv_data)

    test_sl_tensor = tf.constant(test_sl_data)
    test_tv_tensor = tf.constant(test_tv_data)
    test_fl_tensor = tf.constant(test_fl_data)

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
        sl_i = Input(shape=(1,), name='sl', dtype='string')
        sl_layer = StringLookup(name='sl_string_lookup')
        sl_layer.adapt(vocab_data)

        # string lookup alternate
        sa_i = Input(shape=(1,), name='sa', dtype='string')
        sa_layer = StringLookup(name='sa_string_lookup')
        sa_layer.adapt(vocab_data)

        o = Concatenate()([sa_layer(sa_i), sl_layer(sl_i)])
        return [sl_i, sa_i], o

    def inputs_float():
        sl_i = Input(shape=(1,), name='sl', dtype='string')
        sl_layer = StringLookup(name='sl_string_lookup')
        sl_layer.adapt(vocab_data)

        fl_i = Input(shape=(1,), name='fl', dtype='float32')

        o = tf.math.divide(sl_layer(sl_i), 7, name='sl_divide')
        o = Concatenate()([o, fl_i])
        return [sl_i, fl_i], o

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
            {'sa': test_sl_tensor, 'sl': test_sl_tensor}
        ),
        (
            pa.no_float_input, 
            'float_input', 
            inputs_float, 
            {'fl': test_fl_tensor, 'sl': test_sl_tensor}
        )]

    output_mods = [(pa.no_float_out, True, 'float'), 
                   (pa.no_int_out, False, 'int')]

    def model_test_save(test_tensor, check_model):
        check_model.summary()

        print('# Test Input')
        print(test_tensor)

        print('# Prediction')
        start = time.perf_counter()
        print(check_model.predict(test_tensor))
        print(f'{time.perf_counter() - start} elapsed')

        save_dir = f'{check_model.name}_model'
        print(f'Saving model to {save_dir}...')
        check_model.save(save_dir, include_optimizer=False)
        print(f'Saved model to {save_dir}.')

    for gen_params, mod_params in itertools.product(input_generators, output_mods):
        is_disabled, gen_name, generator, tester = gen_params
        if is_disabled:
            continue

        # note variable name overlap
        is_disabled, include_divide, mod_name = mod_params
        if is_disabled:
            continue

        inputs, o = generator()

        o = tf.math.add(o, 1, name='add_one')
        o = tf.math.reduce_prod(o, axis=1, name='reduce_prod')

        if include_divide:
            o = tf.math.divide(o, 7, name='divide',)
            o = tf.cast(o, 'float32', name='cast_f32')

        o = Lambda(lambda o: tf.expand_dims(o, axis=1), name='expand')(o)

        check_model = Model(inputs=inputs, outputs=o, name=f'{gen_name}_{mod_name}')

        print('# Inputs')
        print({i.name: i.dtype for i in check_model.inputs})

        print('# Outputs')
        print({o.name: o.dtype for o in check_model.outputs})

        tester_tensor = {k: tf.expand_dims(v, axis=1) for k, v in tester.items()}
        model_test_save(tester_tensor, check_model)

    if not pa.no_lookup_layer:
        print(test_ll_data)
        str_cols = ['s1', 's2']

        lookup_source = {c: [chr(ord(l) + i) for l in vocab_data] for i, c in enumerate(str_cols)}
        print(lookup_source)

        # explicitly use layers.Input instead of keras.Input
        input_layer = tf.keras.layers.Input(name='lookuplayer_input', shape=(1,), dtype=tf.int32)

        keys = list(range(len(vocab_data)))
        lookup_layers = [LookupLayer(keys, lookup_source[c], default_value='', name=c) for c in str_cols]

        outputs = [lookup_layer(input_layer) for lookup_layer in lookup_layers]
        wrapped_model = Model(inputs=[input_layer], outputs=outputs, name='broken_case_v0_5_0')

        test_tf_tensor = tf.expand_dims(tf.constant(test_ll_data), axis=1)
        model_test_save(test_tf_tensor, wrapped_model)

    if not pa.no_keyed_out:
        class KeyedOut(Model):
            def __init__(self, **kwargs):
                super().__init__(**kwargs)

            def call(self, inputs, training=False):
                copy = inputs.copy()
                return dict(i0_copy=copy['i'], i1_copy=copy['i'])

        ko_model = KeyedOut(name='keyed_out')
        test_tf_tensor = dict(i=tf.expand_dims(tf.constant(test_ll_data), axis=1))
        ko_model.predict(test_tf_tensor)
        model_test_save(test_tf_tensor, ko_model)

    if not pa.no_slow:
        repeat_depth = (1024 * 4) ** 2

        r = Lambda(lambda t: tf.repeat(t, repeat_depth, axis=-1), name='repeater')
        d = Lambda(lambda d: tf.divide(d['x'], d['y']), name='divider')
        m = Lambda(lambda m: tf.multiply(m['x'], m['y']), name='multiplier')
        s = Lambda(lambda t: tf.expand_dims(tf.math.reduce_sum(t, axis=-1), axis=-1), name='summer')

        i_x = tf.keras.layers.Input(name='x', shape=(1,), dtype=tf.float32)
        i_y = tf.keras.layers.Input(name='y', shape=(1,), dtype=tf.float32)

        rx = r(i_x)
        ry = r(i_y)
        # rx / ry
        o = d(dict(x=rx, y=ry))

        for i in range(pa.slow_repeat):
            # rx^2 / ry
            o = m(dict(x=o, y=rx))
            # sum(rx^2/ry)
            so = s(o)
            # rx^2 * sum(rx^2/ry) / ry
            o = m(dict(x=o, y=so))
            # rx^2 * sum(rx^2/ry) / ry^2
            o = d(dict(x=o, y=ry))
            # sum(rx^2 * sum(rx^2/ry) / ry^2)
            so = s(o)
            # [rx^2 * sum(rx^2/ry)] / [ry^2 * sum(rx^2 * sum(rx^2/ry) / ry^2)]
            o = d(dict(x=o, y=so))

        o = s(o)
        o = Lambda(lambda t: t, name='output')(o)

        slow_model = Model(inputs=[i_x, i_y], outputs=[o], name='slow')

        test_tf_tensor = dict(x=tf.constant([[4], [8]]), y=tf.constant([[5], [9]]))
        model_test_save(test_tf_tensor, slow_model)

    print('Done.')

