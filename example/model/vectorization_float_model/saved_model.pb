æ²
ē
:
Add
x"T
y"T
z"T"
Ttype:
2	
B
AddV2
x"T
y"T
z"T"
Ttype:
2	
K
Bincount
arr
size
weights"T	
bins"T"
Ttype:
2	
N
Cast	
x"SrcT	
y"DstT"
SrcTtype"
DstTtype"
Truncatebool( 
h
ConcatV2
values"T*N
axis"Tidx
output"T"
Nint(0"	
Ttype"
Tidxtype0:
2	
8
Const
output"dtype"
valuetensor"
dtypetype

Cumsum
x"T
axis"Tidx
out"T"
	exclusivebool( "
reversebool( " 
Ttype:
2	"
Tidxtype0:
2	
W

ExpandDims

input"T
dim"Tdim
output"T"	
Ttype"
Tdimtype0:
2	
=
Greater
x"T
y"T
z
"
Ttype:
2	
.
Identity

input"T
output"T"	
Ttype
l
LookupTableExportV2
table_handle
keys"Tkeys
values"Tvalues"
Tkeystype"
Tvaluestype
w
LookupTableFindV2
table_handle
keys"Tin
default_value"Tout
values"Tout"
Tintype"
Touttype
b
LookupTableImportV2
table_handle
keys"Tin
values"Tout"
Tintype"
Touttype

Max

input"T
reduction_indices"Tidx
output"T"
	keep_dimsbool( " 
Ttype:
2	"
Tidxtype0:
2	
:
Maximum
x"T
y"T
z"T"
Ttype:

2	
e
MergeV2Checkpoints
checkpoint_prefixes
destination_prefix"
delete_old_dirsbool(
:
Minimum
x"T
y"T
z"T"
Ttype:

2	
=
Mul
x"T
y"T
z"T"
Ttype:
2	
Ø
MutableHashTableV2
table_handle"
	containerstring "
shared_namestring "!
use_node_name_sharingbool( "
	key_dtypetype"
value_dtypetype

NoOp
M
Pack
values"T*N
output"T"
Nint(0"	
Ttype"
axisint 
³
PartitionedCall
args2Tin
output2Tout"
Tin
list(type)("
Tout
list(type)("	
ffunc"
configstring "
config_protostring "
executor_typestring 
C
Placeholder
output"dtype"
dtypetype"
shapeshape:

Prod

input"T
reduction_indices"Tidx
output"T"
	keep_dimsbool( " 
Ttype:
2	"
Tidxtype0:
2	

RaggedTensorToTensor
shape"Tshape
values"T
default_value"T:
row_partition_tensors"Tindex*num_row_partition_tensors
result"T"	
Ttype"
Tindextype:
2	"
Tshapetype:
2	"$
num_row_partition_tensorsint(0"#
row_partition_typeslist(string)
>
RealDiv
x"T
y"T
z"T"
Ttype:
2	
o
	RestoreV2

prefix
tensor_names
shape_and_slices
tensors2dtypes"
dtypes
list(type)(0
l
SaveV2

prefix
tensor_names
shape_and_slices
tensors2dtypes"
dtypes
list(type)(0
?
Select
	condition

t"T
e"T
output"T"	
Ttype
P
Shape

input"T
output"out_type"	
Ttype"
out_typetype0:
2	
H
ShardedFilename
basename	
shard

num_shards
filename
N
Squeeze

input"T
output"T"	
Ttype"
squeeze_dims	list(int)
 (
¾
StatefulPartitionedCall
args2Tin
output2Tout"
Tin
list(type)("
Tout
list(type)("	
ffunc"
configstring "
config_protostring "
executor_typestring 
@
StaticRegexFullMatch	
input

output
"
patternstring
m
StaticRegexReplace	
input

output"
patternstring"
rewritestring"
replace_globalbool(
ö
StridedSlice

input"T
begin"Index
end"Index
strides"Index
output"T"	
Ttype"
Indextype:
2	"

begin_maskint "
end_maskint "
ellipsis_maskint "
new_axis_maskint "
shrink_axis_maskint 
N

StringJoin
inputs*N

output"
Nint(0"
	separatorstring 
<
StringLower	
input

output"
encodingstring 
e
StringSplitV2	
input
sep
indices	

values	
shape	"
maxsplitint’’’’’’’’’"serve*2.4.32v2.4.2-142-g72bb4c22adb8Ų	

sl_string_lookup_index_tableMutableHashTableV2*
_output_shapes
: *
	key_dtype0*
shared_name
table_73*
value_dtype0	

string_lookup_index_tableMutableHashTableV2*
_output_shapes
: *
	key_dtype0*
shared_name	table_6*
value_dtype0	
G
ConstConst*
_output_shapes
: *
dtype0	*
value	B	 R
I
Const_1Const*
_output_shapes
: *
dtype0	*
value	B	 R
č
PartitionedCallPartitionedCall*	
Tin
 *
Tout
2*
_collective_manager_ids
 *
_output_shapes
: * 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *"
fR
__inference_<lambda>_1067
ź
PartitionedCall_1PartitionedCall*	
Tin
 *
Tout
2*
_collective_manager_ids
 *
_output_shapes
: * 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *"
fR
__inference_<lambda>_1072
2
NoOpNoOp^PartitionedCall^PartitionedCall_1
ė
Ksl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2LookupTableExportV2sl_string_lookup_index_table*
Tkeys0*
Tvalues0	*/
_class%
#!loc:@sl_string_lookup_index_table*
_output_shapes

::
ā
Hstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2LookupTableExportV2string_lookup_index_table*
Tkeys0*
Tvalues0	*,
_class"
 loc:@string_lookup_index_table*
_output_shapes

::
­
Const_2Const"/device:CPU:0*
_output_shapes
: *
dtype0*ę
valueÜBŁ BŅ

layer-0
layer-1
layer_with_weights-0
layer-2
layer_with_weights-1
layer-3
layer-4
layer-5
layer-6
layer-7
	layer-8

layer-9
regularization_losses
trainable_variables
	variables
	keras_api

signatures
 
 
=
state_variables
_index_lookup_layer
	keras_api
0
state_variables

_table
	keras_api
R
regularization_losses
trainable_variables
	variables
	keras_api

	keras_api

	keras_api

	keras_api

	keras_api
R
regularization_losses
trainable_variables
 	variables
!	keras_api
 
 
 
­
"metrics
#layer_metrics
$non_trainable_variables
%layer_regularization_losses

&layers
regularization_losses
trainable_variables
	variables
 
 
0
'state_variables

(_table
)	keras_api
 
 
86
table-layer_with_weights-1/_table/.ATTRIBUTES/table
 
 
 
 
­
*metrics
+layer_metrics
,non_trainable_variables
-layer_regularization_losses

.layers
regularization_losses
trainable_variables
	variables
 
 
 
 
 
 
 
­
/metrics
0layer_metrics
1non_trainable_variables
2layer_regularization_losses

3layers
regularization_losses
trainable_variables
 	variables
 
 
 
 
F
0
1
2
3
4
5
6
7
	8

9
 
LJ
tableAlayer_with_weights-0/_index_lookup_layer/_table/.ATTRIBUTES/table
 
 
 
 
 
 
 
 
 
 
 
u
serving_default_slPlaceholder*'
_output_shapes
:’’’’’’’’’*
dtype0*
shape:’’’’’’’’’
u
serving_default_tvPlaceholder*'
_output_shapes
:’’’’’’’’’*
dtype0*
shape:’’’’’’’’’

StatefulPartitionedCallStatefulPartitionedCallserving_default_slserving_default_tvstring_lookup_index_tableConstsl_string_lookup_index_tableConst_1*
Tin

2		*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 **
f%R#
!__inference_signature_wrapper_729
O
saver_filenamePlaceholder*
_output_shapes
: *
dtype0*
shape: 
Ń
StatefulPartitionedCall_1StatefulPartitionedCallsaver_filenameKsl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2Msl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2:1Hstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2Jstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2:1Const_2*
Tin

2		*
Tout
2*
_collective_manager_ids
 *
_output_shapes
: * 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *&
f!R
__inference__traced_save_1110
Ļ
StatefulPartitionedCall_2StatefulPartitionedCallsaver_filenamesl_string_lookup_index_tablestring_lookup_index_table*
Tin
2*
Tout
2*
_collective_manager_ids
 *
_output_shapes
: * 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *)
f$R"
 __inference__traced_restore_1126Õ³	
µ
[
?__inference_expand_layer_call_and_return_conditional_losses_472

inputs
identityb
ExpandDims/dimConst*
_output_shapes
: *
dtype0*
value	B :2
ExpandDims/dimy

ExpandDims
ExpandDimsinputsExpandDims/dim:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

ExpandDimsg
IdentityIdentityExpandDims:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*"
_input_shapes
:’’’’’’’’’:K G
#
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs

Ź
__inference__traced_save_1110
file_prefixV
Rsavev2_sl_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2X
Tsavev2_sl_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2_1	S
Osavev2_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2U
Qsavev2_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2_1	
savev2_const_2

identity_1¢MergeV2Checkpoints
StaticRegexFullMatchStaticRegexFullMatchfile_prefix"/device:CPU:**
_output_shapes
: *
pattern
^s3://.*2
StaticRegexFullMatchc
ConstConst"/device:CPU:**
_output_shapes
: *
dtype0*
valueB B.part2
Constl
Const_1Const"/device:CPU:**
_output_shapes
: *
dtype0*
valueB B
_temp/part2	
Const_1
SelectSelectStaticRegexFullMatch:output:0Const:output:0Const_1:output:0"/device:CPU:**
T0*
_output_shapes
: 2
Selectt

StringJoin
StringJoinfile_prefixSelect:output:0"/device:CPU:**
N*
_output_shapes
: 2

StringJoinZ

num_shardsConst*
_output_shapes
: *
dtype0*
value	B :2

num_shards
ShardedFilename/shardConst"/device:CPU:0*
_output_shapes
: *
dtype0*
value	B : 2
ShardedFilename/shard¦
ShardedFilenameShardedFilenameStringJoin:output:0ShardedFilename/shard:output:0num_shards:output:0"/device:CPU:0*
_output_shapes
: 2
ShardedFilename
SaveV2/tensor_namesConst"/device:CPU:0*
_output_shapes
:*
dtype0*Æ
value„B¢B2layer_with_weights-1/_table/.ATTRIBUTES/table-keysB4layer_with_weights-1/_table/.ATTRIBUTES/table-valuesBFlayer_with_weights-0/_index_lookup_layer/_table/.ATTRIBUTES/table-keysBHlayer_with_weights-0/_index_lookup_layer/_table/.ATTRIBUTES/table-valuesB_CHECKPOINTABLE_OBJECT_GRAPH2
SaveV2/tensor_names
SaveV2/shape_and_slicesConst"/device:CPU:0*
_output_shapes
:*
dtype0*
valueBB B B B B 2
SaveV2/shape_and_slices
SaveV2SaveV2ShardedFilename:filename:0SaveV2/tensor_names:output:0 SaveV2/shape_and_slices:output:0Rsavev2_sl_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2Tsavev2_sl_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2_1Osavev2_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2Qsavev2_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2_1savev2_const_2"/device:CPU:0*
_output_shapes
 *
dtypes	
2		2
SaveV2ŗ
&MergeV2Checkpoints/checkpoint_prefixesPackShardedFilename:filename:0^SaveV2"/device:CPU:0*
N*
T0*
_output_shapes
:2(
&MergeV2Checkpoints/checkpoint_prefixes”
MergeV2CheckpointsMergeV2Checkpoints/MergeV2Checkpoints/checkpoint_prefixes:output:0file_prefix"/device:CPU:0*
_output_shapes
 2
MergeV2Checkpointsr
IdentityIdentityfile_prefix^MergeV2Checkpoints"/device:CPU:0*
T0*
_output_shapes
: 2

Identitym

Identity_1IdentityIdentity:output:0^MergeV2Checkpoints*
T0*
_output_shapes
: 2

Identity_1"!

identity_1Identity_1:output:0*'
_input_shapes
: ::::: 2(
MergeV2CheckpointsMergeV2Checkpoints:C ?

_output_shapes
: 
%
_user_specified_namefile_prefix:

_output_shapes
::

_output_shapes
::

_output_shapes
::

_output_shapes
::

_output_shapes
: 

)
__inference_<lambda>_1072
identityS
ConstConst*
_output_shapes
: *
dtype0*
valueB
 *  ?2
ConstQ
IdentityIdentityConst:output:0*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 
µ
[
?__inference_expand_layer_call_and_return_conditional_losses_478

inputs
identityb
ExpandDims/dimConst*
_output_shapes
: *
dtype0*
value	B :2
ExpandDims/dimy

ExpandDims
ExpandDimsinputsExpandDims/dim:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

ExpandDimsg
IdentityIdentityExpandDims:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*"
_input_shapes
:’’’’’’’’’:K G
#
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs

*
__inference__destroyer_993
identityP
ConstConst*
_output_shapes
: *
dtype0*
value	B :2
ConstQ
IdentityIdentityConst:output:0*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 
ē

 __inference__traced_restore_1126
file_prefix_
[sl_string_lookup_index_table_table_restore_lookuptableimportv2_sl_string_lookup_index_tableY
Ustring_lookup_index_table_table_restore_lookuptableimportv2_string_lookup_index_table

identity_1¢>sl_string_lookup_index_table_table_restore/LookupTableImportV2¢;string_lookup_index_table_table_restore/LookupTableImportV2£
RestoreV2/tensor_namesConst"/device:CPU:0*
_output_shapes
:*
dtype0*Æ
value„B¢B2layer_with_weights-1/_table/.ATTRIBUTES/table-keysB4layer_with_weights-1/_table/.ATTRIBUTES/table-valuesBFlayer_with_weights-0/_index_lookup_layer/_table/.ATTRIBUTES/table-keysBHlayer_with_weights-0/_index_lookup_layer/_table/.ATTRIBUTES/table-valuesB_CHECKPOINTABLE_OBJECT_GRAPH2
RestoreV2/tensor_names
RestoreV2/shape_and_slicesConst"/device:CPU:0*
_output_shapes
:*
dtype0*
valueBB B B B B 2
RestoreV2/shape_and_slicesÄ
	RestoreV2	RestoreV2file_prefixRestoreV2/tensor_names:output:0#RestoreV2/shape_and_slices:output:0"/device:CPU:0*(
_output_shapes
:::::*
dtypes	
2		2
	RestoreV2ü
>sl_string_lookup_index_table_table_restore/LookupTableImportV2LookupTableImportV2[sl_string_lookup_index_table_table_restore_lookuptableimportv2_sl_string_lookup_index_tableRestoreV2:tensors:0RestoreV2:tensors:1*	
Tin0*

Tout0	*/
_class%
#!loc:@sl_string_lookup_index_table*
_output_shapes
 2@
>sl_string_lookup_index_table_table_restore/LookupTableImportV2ķ
;string_lookup_index_table_table_restore/LookupTableImportV2LookupTableImportV2Ustring_lookup_index_table_table_restore_lookuptableimportv2_string_lookup_index_tableRestoreV2:tensors:2RestoreV2:tensors:3*	
Tin0*

Tout0	*,
_class"
 loc:@string_lookup_index_table*
_output_shapes
 2=
;string_lookup_index_table_table_restore/LookupTableImportV29
NoOpNoOp"/device:CPU:0*
_output_shapes
 2
NoOpć
IdentityIdentityfile_prefix^NoOp?^sl_string_lookup_index_table_table_restore/LookupTableImportV2<^string_lookup_index_table_table_restore/LookupTableImportV2"/device:CPU:0*
T0*
_output_shapes
: 2

Identity×

Identity_1IdentityIdentity:output:0?^sl_string_lookup_index_table_table_restore/LookupTableImportV2<^string_lookup_index_table_table_restore/LookupTableImportV2*
T0*
_output_shapes
: 2

Identity_1"!

identity_1Identity_1:output:0*
_input_shapes

: ::2
>sl_string_lookup_index_table_table_restore/LookupTableImportV2>sl_string_lookup_index_table_table_restore/LookupTableImportV22z
;string_lookup_index_table_table_restore/LookupTableImportV2;string_lookup_index_table_table_restore/LookupTableImportV2:C ?

_output_shapes
: 
%
_user_specified_namefile_prefix:51
/
_class%
#!loc:@sl_string_lookup_index_table:2.
,
_class"
 loc:@string_lookup_index_table
»
U
)__inference_concatenate_layer_call_fn_956
inputs_0	
inputs_1	
identity	Ų
PartitionedCallPartitionedCallinputs_0inputs_1*
Tin
2		*
Tout
2	*
_collective_manager_ids
 *0
_output_shapes
:’’’’’’’’’’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *M
fHRF
D__inference_concatenate_layer_call_and_return_conditional_losses_4482
PartitionedCallu
IdentityIdentityPartitionedCall:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2

Identity"
identityIdentity:output:0*B
_input_shapes1
/:’’’’’’’’’’’’’’’’’’:’’’’’’’’’:Z V
0
_output_shapes
:’’’’’’’’’’’’’’’’’’
"
_user_specified_name
inputs/0:QM
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/1

@
$__inference_expand_layer_call_fn_978

inputs
identity½
PartitionedCallPartitionedCallinputs*
Tin
2*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *H
fCRA
?__inference_expand_layer_call_and_return_conditional_losses_4782
PartitionedCalll
IdentityIdentityPartitionedCall:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*"
_input_shapes
:’’’’’’’’’:K G
#
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs

)
__inference_<lambda>_1067
identityS
ConstConst*
_output_shapes
: *
dtype0*
valueB
 *  ?2
ConstQ
IdentityIdentityConst:output:0*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 
Ą
Ø
1__inference_vectorization_float_layer_call_fn_713
tv
sl
unknown
	unknown_0	
	unknown_1
	unknown_2	
identity¢StatefulPartitionedCall
StatefulPartitionedCallStatefulPartitionedCalltvslunknown	unknown_0	unknown_1	unknown_2*
Tin

2		*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *U
fPRN
L__inference_vectorization_float_layer_call_and_return_conditional_losses_7022
StatefulPartitionedCall
IdentityIdentity StatefulPartitionedCall:output:0^StatefulPartitionedCall*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 22
StatefulPartitionedCallStatefulPartitionedCall:K G
'
_output_shapes
:’’’’’’’’’

_user_specified_nametv:KG
'
_output_shapes
:’’’’’’’’’

_user_specified_namesl:

_output_shapes
: :

_output_shapes
: 
ō
Ü
L__inference_vectorization_float_layer_call_and_return_conditional_losses_624

inputs
inputs_1]
Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle^
Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	J
Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleK
Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	
identity¢9sl_string_lookup/None_lookup_table_find/LookupTableFindV2¢Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
!tv_text_vectorization/StringLowerStringLowerinputs*'
_output_shapes
:’’’’’’’’’2#
!tv_text_vectorization/StringLower
(tv_text_vectorization/StaticRegexReplaceStaticRegexReplace*tv_text_vectorization/StringLower:output:0*'
_output_shapes
:’’’’’’’’’*6
pattern+)[!"#$%&()\*\+,-\./:;<=>?@\[\\\]^_`{|}~\']*
rewrite 2*
(tv_text_vectorization/StaticRegexReplaceŹ
tv_text_vectorization/SqueezeSqueeze1tv_text_vectorization/StaticRegexReplace:output:0*
T0*#
_output_shapes
:’’’’’’’’’*
squeeze_dims

’’’’’’’’’2
tv_text_vectorization/Squeeze
'tv_text_vectorization/StringSplit/ConstConst*
_output_shapes
: *
dtype0*
valueB B 2)
'tv_text_vectorization/StringSplit/Const
/tv_text_vectorization/StringSplit/StringSplitV2StringSplitV2&tv_text_vectorization/Squeeze:output:00tv_text_vectorization/StringSplit/Const:output:0*<
_output_shapes*
(:’’’’’’’’’:’’’’’’’’’:21
/tv_text_vectorization/StringSplit/StringSplitV2æ
5tv_text_vectorization/StringSplit/strided_slice/stackConst*
_output_shapes
:*
dtype0*
valueB"        27
5tv_text_vectorization/StringSplit/strided_slice/stackĆ
7tv_text_vectorization/StringSplit/strided_slice/stack_1Const*
_output_shapes
:*
dtype0*
valueB"       29
7tv_text_vectorization/StringSplit/strided_slice/stack_1Ć
7tv_text_vectorization/StringSplit/strided_slice/stack_2Const*
_output_shapes
:*
dtype0*
valueB"      29
7tv_text_vectorization/StringSplit/strided_slice/stack_2ę
/tv_text_vectorization/StringSplit/strided_sliceStridedSlice9tv_text_vectorization/StringSplit/StringSplitV2:indices:0>tv_text_vectorization/StringSplit/strided_slice/stack:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_1:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_2:output:0*
Index0*
T0	*#
_output_shapes
:’’’’’’’’’*

begin_mask*
end_mask*
shrink_axis_mask21
/tv_text_vectorization/StringSplit/strided_slice¼
7tv_text_vectorization/StringSplit/strided_slice_1/stackConst*
_output_shapes
:*
dtype0*
valueB: 29
7tv_text_vectorization/StringSplit/strided_slice_1/stackĄ
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Ą
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2æ
1tv_text_vectorization/StringSplit/strided_slice_1StridedSlice7tv_text_vectorization/StringSplit/StringSplitV2:shape:0@tv_text_vectorization/StringSplit/strided_slice_1/stack:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_1:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_2:output:0*
Index0*
T0	*
_output_shapes
: *
shrink_axis_mask23
1tv_text_vectorization/StringSplit/strided_slice_1³
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CastCast8tv_text_vectorization/StringSplit/strided_slice:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2Z
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast¬
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Cast:tv_text_vectorization/StringSplit/strided_slice_1:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Ō
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ShapeShape\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0*
T0*
_output_shapes
:2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstConst*
_output_shapes
:*
dtype0*
valueB: 2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstÉ
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ProdProdktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const:output:0*
T0*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yConst*
_output_shapes
: *
dtype0*
value	B : 2h
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yÕ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/GreaterGreaterjtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod:output:0otv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y:output:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greaterč
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/CastCasthtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater:z:0*

DstT0*

SrcT0
*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1Const*
_output_shapes
:*
dtype0*
valueB: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaxMax\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yConst*
_output_shapes
: *
dtype0*
value	B :2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yĘ
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/addAddV2itv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mulMuletv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add:z:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul¾
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumMaximum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumĀ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MinimumMinimum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2Const*
_output_shapes
: *
dtype0	*
valueB	 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2æ
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/BincountBincount\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum:z:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2g
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisČ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CumsumCumsumltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount:bins:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0Const*
_output_shapes
:*
dtype0	*
valueB	R 2e
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisµ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatConcatV2ltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0:output:0`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum:out:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis:output:0*
N*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle8tv_text_vectorization/StringSplit/StringSplitV2:values:0Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*#
_output_shapes
:’’’’’’’’’2N
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
5tv_text_vectorization/string_lookup/assert_equal/NoOpNoOp*
_output_shapes
 27
5tv_text_vectorization/string_lookup/assert_equal/NoOpķ
,tv_text_vectorization/string_lookup/IdentityIdentityUtv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
T0	*#
_output_shapes
:’’’’’’’’’2.
,tv_text_vectorization/string_lookup/Identity’
.tv_text_vectorization/string_lookup/Identity_1Identityctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat:output:0*
T0	*#
_output_shapes
:’’’’’’’’’20
.tv_text_vectorization/string_lookup/Identity_1Ŗ
2tv_text_vectorization/RaggedToTensor/default_valueConst*
_output_shapes
: *
dtype0	*
value	B	 R 24
2tv_text_vectorization/RaggedToTensor/default_value£
*tv_text_vectorization/RaggedToTensor/ConstConst*
_output_shapes
: *
dtype0	*
valueB	 R
’’’’’’’’’2,
*tv_text_vectorization/RaggedToTensor/Const
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorRaggedTensorToTensor3tv_text_vectorization/RaggedToTensor/Const:output:05tv_text_vectorization/string_lookup/Identity:output:0;tv_text_vectorization/RaggedToTensor/default_value:output:07tv_text_vectorization/string_lookup/Identity_1:output:0*
T0	*
Tindex0	*
Tshape0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’*
num_row_partition_tensors*%
row_partition_types

ROW_SPLITS2;
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensor
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleinputs_1Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*'
_output_shapes
:’’’’’’’’’2;
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2ä
concatenate/PartitionedCallPartitionedCallBtv_text_vectorization/RaggedToTensor/RaggedTensorToTensor:result:0Bsl_string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
Tin
2		*
Tout
2	*
_collective_manager_ids
 *0
_output_shapes
:’’’’’’’’’’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *M
fHRF
D__inference_concatenate_layer_call_and_return_conditional_losses_4482
concatenate/PartitionedCallh
tf.math.add/Add/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.add/Add/y¦
tf.math.add/AddAdd$concatenate/PartitionedCall:output:0tf.math.add/Add/y:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
tf.math.add/Add
*tf.math.reduce_prod/Prod/reduction_indicesConst*
_output_shapes
: *
dtype0*
value	B :2,
*tf.math.reduce_prod/Prod/reduction_indices“
tf.math.reduce_prod/ProdProdtf.math.add/Add:z:03tf.math.reduce_prod/Prod/reduction_indices:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2
tf.math.reduce_prod/Prodv
tf.math.divide/truediv/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.divide/truediv/y¢
tf.math.divide/truediv/CastCast!tf.math.reduce_prod/Prod:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv/Cast
tf.math.divide/truediv/Cast_1Cast!tf.math.divide/truediv/y:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2
tf.math.divide/truediv/Cast_1­
tf.math.divide/truedivRealDivtf.math.divide/truediv/Cast:y:0!tf.math.divide/truediv/Cast_1:y:0*
T0*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv}
tf.cast/CastCasttf.math.divide/truediv:z:0*

DstT0*

SrcT0*#
_output_shapes
:’’’’’’’’’2
tf.cast/CastÕ
expand/PartitionedCallPartitionedCalltf.cast/Cast:y:0*
Tin
2*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *H
fCRA
?__inference_expand_layer_call_and_return_conditional_losses_4722
expand/PartitionedCallž
IdentityIdentityexpand/PartitionedCall:output:0:^sl_string_lookup/None_lookup_table_find/LookupTableFindV2M^tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 2v
9sl_string_lookup/None_lookup_table_find/LookupTableFindV29sl_string_lookup/None_lookup_table_find/LookupTableFindV22
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:O K
'
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs:OK
'
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs:

_output_shapes
: :

_output_shapes
: 
ä
“
1__inference_vectorization_float_layer_call_fn_943
inputs_0
inputs_1
unknown
	unknown_0	
	unknown_1
	unknown_2	
identity¢StatefulPartitionedCall
StatefulPartitionedCallStatefulPartitionedCallinputs_0inputs_1unknown	unknown_0	unknown_1	unknown_2*
Tin

2		*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *U
fPRN
L__inference_vectorization_float_layer_call_and_return_conditional_losses_7022
StatefulPartitionedCall
IdentityIdentity StatefulPartitionedCall:output:0^StatefulPartitionedCall*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 22
StatefulPartitionedCallStatefulPartitionedCall:Q M
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/0:QM
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/1:

_output_shapes
: :

_output_shapes
: 
Ż
p
D__inference_concatenate_layer_call_and_return_conditional_losses_950
inputs_0	
inputs_1	
identity	\
concat/axisConst*
_output_shapes
: *
dtype0*
value	B :2
concat/axis
concatConcatV2inputs_0inputs_1concat/axis:output:0*
N*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
concatl
IdentityIdentityconcat:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2

Identity"
identityIdentity:output:0*B
_input_shapes1
/:’’’’’’’’’’’’’’’’’’:’’’’’’’’’:Z V
0
_output_shapes
:’’’’’’’’’’’’’’’’’’
"
_user_specified_name
inputs/0:QM
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/1
×
Ž
L__inference_vectorization_float_layer_call_and_return_conditional_losses_915
inputs_0
inputs_1]
Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle^
Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	J
Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleK
Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	
identity¢9sl_string_lookup/None_lookup_table_find/LookupTableFindV2¢Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
!tv_text_vectorization/StringLowerStringLowerinputs_0*'
_output_shapes
:’’’’’’’’’2#
!tv_text_vectorization/StringLower
(tv_text_vectorization/StaticRegexReplaceStaticRegexReplace*tv_text_vectorization/StringLower:output:0*'
_output_shapes
:’’’’’’’’’*6
pattern+)[!"#$%&()\*\+,-\./:;<=>?@\[\\\]^_`{|}~\']*
rewrite 2*
(tv_text_vectorization/StaticRegexReplaceŹ
tv_text_vectorization/SqueezeSqueeze1tv_text_vectorization/StaticRegexReplace:output:0*
T0*#
_output_shapes
:’’’’’’’’’*
squeeze_dims

’’’’’’’’’2
tv_text_vectorization/Squeeze
'tv_text_vectorization/StringSplit/ConstConst*
_output_shapes
: *
dtype0*
valueB B 2)
'tv_text_vectorization/StringSplit/Const
/tv_text_vectorization/StringSplit/StringSplitV2StringSplitV2&tv_text_vectorization/Squeeze:output:00tv_text_vectorization/StringSplit/Const:output:0*<
_output_shapes*
(:’’’’’’’’’:’’’’’’’’’:21
/tv_text_vectorization/StringSplit/StringSplitV2æ
5tv_text_vectorization/StringSplit/strided_slice/stackConst*
_output_shapes
:*
dtype0*
valueB"        27
5tv_text_vectorization/StringSplit/strided_slice/stackĆ
7tv_text_vectorization/StringSplit/strided_slice/stack_1Const*
_output_shapes
:*
dtype0*
valueB"       29
7tv_text_vectorization/StringSplit/strided_slice/stack_1Ć
7tv_text_vectorization/StringSplit/strided_slice/stack_2Const*
_output_shapes
:*
dtype0*
valueB"      29
7tv_text_vectorization/StringSplit/strided_slice/stack_2ę
/tv_text_vectorization/StringSplit/strided_sliceStridedSlice9tv_text_vectorization/StringSplit/StringSplitV2:indices:0>tv_text_vectorization/StringSplit/strided_slice/stack:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_1:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_2:output:0*
Index0*
T0	*#
_output_shapes
:’’’’’’’’’*

begin_mask*
end_mask*
shrink_axis_mask21
/tv_text_vectorization/StringSplit/strided_slice¼
7tv_text_vectorization/StringSplit/strided_slice_1/stackConst*
_output_shapes
:*
dtype0*
valueB: 29
7tv_text_vectorization/StringSplit/strided_slice_1/stackĄ
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Ą
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2æ
1tv_text_vectorization/StringSplit/strided_slice_1StridedSlice7tv_text_vectorization/StringSplit/StringSplitV2:shape:0@tv_text_vectorization/StringSplit/strided_slice_1/stack:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_1:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_2:output:0*
Index0*
T0	*
_output_shapes
: *
shrink_axis_mask23
1tv_text_vectorization/StringSplit/strided_slice_1³
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CastCast8tv_text_vectorization/StringSplit/strided_slice:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2Z
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast¬
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Cast:tv_text_vectorization/StringSplit/strided_slice_1:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Ō
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ShapeShape\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0*
T0*
_output_shapes
:2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstConst*
_output_shapes
:*
dtype0*
valueB: 2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstÉ
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ProdProdktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const:output:0*
T0*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yConst*
_output_shapes
: *
dtype0*
value	B : 2h
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yÕ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/GreaterGreaterjtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod:output:0otv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y:output:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greaterč
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/CastCasthtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater:z:0*

DstT0*

SrcT0
*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1Const*
_output_shapes
:*
dtype0*
valueB: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaxMax\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yConst*
_output_shapes
: *
dtype0*
value	B :2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yĘ
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/addAddV2itv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mulMuletv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add:z:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul¾
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumMaximum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumĀ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MinimumMinimum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2Const*
_output_shapes
: *
dtype0	*
valueB	 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2æ
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/BincountBincount\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum:z:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2g
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisČ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CumsumCumsumltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount:bins:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0Const*
_output_shapes
:*
dtype0	*
valueB	R 2e
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisµ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatConcatV2ltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0:output:0`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum:out:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis:output:0*
N*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle8tv_text_vectorization/StringSplit/StringSplitV2:values:0Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*#
_output_shapes
:’’’’’’’’’2N
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
5tv_text_vectorization/string_lookup/assert_equal/NoOpNoOp*
_output_shapes
 27
5tv_text_vectorization/string_lookup/assert_equal/NoOpķ
,tv_text_vectorization/string_lookup/IdentityIdentityUtv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
T0	*#
_output_shapes
:’’’’’’’’’2.
,tv_text_vectorization/string_lookup/Identity’
.tv_text_vectorization/string_lookup/Identity_1Identityctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat:output:0*
T0	*#
_output_shapes
:’’’’’’’’’20
.tv_text_vectorization/string_lookup/Identity_1Ŗ
2tv_text_vectorization/RaggedToTensor/default_valueConst*
_output_shapes
: *
dtype0	*
value	B	 R 24
2tv_text_vectorization/RaggedToTensor/default_value£
*tv_text_vectorization/RaggedToTensor/ConstConst*
_output_shapes
: *
dtype0	*
valueB	 R
’’’’’’’’’2,
*tv_text_vectorization/RaggedToTensor/Const
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorRaggedTensorToTensor3tv_text_vectorization/RaggedToTensor/Const:output:05tv_text_vectorization/string_lookup/Identity:output:0;tv_text_vectorization/RaggedToTensor/default_value:output:07tv_text_vectorization/string_lookup/Identity_1:output:0*
T0	*
Tindex0	*
Tshape0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’*
num_row_partition_tensors*%
row_partition_types

ROW_SPLITS2;
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensor
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleinputs_1Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*'
_output_shapes
:’’’’’’’’’2;
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2t
concatenate/concat/axisConst*
_output_shapes
: *
dtype0*
value	B :2
concatenate/concat/axis¢
concatenate/concatConcatV2Btv_text_vectorization/RaggedToTensor/RaggedTensorToTensor:result:0Bsl_string_lookup/None_lookup_table_find/LookupTableFindV2:values:0 concatenate/concat/axis:output:0*
N*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
concatenate/concath
tf.math.add/Add/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.add/Add/y
tf.math.add/AddAddconcatenate/concat:output:0tf.math.add/Add/y:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
tf.math.add/Add
*tf.math.reduce_prod/Prod/reduction_indicesConst*
_output_shapes
: *
dtype0*
value	B :2,
*tf.math.reduce_prod/Prod/reduction_indices“
tf.math.reduce_prod/ProdProdtf.math.add/Add:z:03tf.math.reduce_prod/Prod/reduction_indices:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2
tf.math.reduce_prod/Prodv
tf.math.divide/truediv/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.divide/truediv/y¢
tf.math.divide/truediv/CastCast!tf.math.reduce_prod/Prod:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv/Cast
tf.math.divide/truediv/Cast_1Cast!tf.math.divide/truediv/y:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2
tf.math.divide/truediv/Cast_1­
tf.math.divide/truedivRealDivtf.math.divide/truediv/Cast:y:0!tf.math.divide/truediv/Cast_1:y:0*
T0*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv}
tf.cast/CastCasttf.math.divide/truediv:z:0*

DstT0*

SrcT0*#
_output_shapes
:’’’’’’’’’2
tf.cast/Castp
expand/ExpandDims/dimConst*
_output_shapes
: *
dtype0*
value	B :2
expand/ExpandDims/dim
expand/ExpandDims
ExpandDimstf.cast/Cast:y:0expand/ExpandDims/dim:output:0*
T0*'
_output_shapes
:’’’’’’’’’2
expand/ExpandDimsł
IdentityIdentityexpand/ExpandDims:output:0:^sl_string_lookup/None_lookup_table_find/LookupTableFindV2M^tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 2v
9sl_string_lookup/None_lookup_table_find/LookupTableFindV29sl_string_lookup/None_lookup_table_find/LookupTableFindV22
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:Q M
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/0:QM
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/1:

_output_shapes
: :

_output_shapes
: 

,
__inference__initializer_988
identityP
ConstConst*
_output_shapes
: *
dtype0*
value	B :2
ConstQ
IdentityIdentityConst:output:0*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 
µ
[
?__inference_expand_layer_call_and_return_conditional_losses_968

inputs
identityb
ExpandDims/dimConst*
_output_shapes
: *
dtype0*
value	B :2
ExpandDims/dimy

ExpandDims
ExpandDimsinputsExpandDims/dim:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

ExpandDimsg
IdentityIdentityExpandDims:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*"
_input_shapes
:’’’’’’’’’:K G
#
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs
×
Ž
L__inference_vectorization_float_layer_call_and_return_conditional_losses_849
inputs_0
inputs_1]
Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle^
Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	J
Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleK
Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	
identity¢9sl_string_lookup/None_lookup_table_find/LookupTableFindV2¢Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
!tv_text_vectorization/StringLowerStringLowerinputs_0*'
_output_shapes
:’’’’’’’’’2#
!tv_text_vectorization/StringLower
(tv_text_vectorization/StaticRegexReplaceStaticRegexReplace*tv_text_vectorization/StringLower:output:0*'
_output_shapes
:’’’’’’’’’*6
pattern+)[!"#$%&()\*\+,-\./:;<=>?@\[\\\]^_`{|}~\']*
rewrite 2*
(tv_text_vectorization/StaticRegexReplaceŹ
tv_text_vectorization/SqueezeSqueeze1tv_text_vectorization/StaticRegexReplace:output:0*
T0*#
_output_shapes
:’’’’’’’’’*
squeeze_dims

’’’’’’’’’2
tv_text_vectorization/Squeeze
'tv_text_vectorization/StringSplit/ConstConst*
_output_shapes
: *
dtype0*
valueB B 2)
'tv_text_vectorization/StringSplit/Const
/tv_text_vectorization/StringSplit/StringSplitV2StringSplitV2&tv_text_vectorization/Squeeze:output:00tv_text_vectorization/StringSplit/Const:output:0*<
_output_shapes*
(:’’’’’’’’’:’’’’’’’’’:21
/tv_text_vectorization/StringSplit/StringSplitV2æ
5tv_text_vectorization/StringSplit/strided_slice/stackConst*
_output_shapes
:*
dtype0*
valueB"        27
5tv_text_vectorization/StringSplit/strided_slice/stackĆ
7tv_text_vectorization/StringSplit/strided_slice/stack_1Const*
_output_shapes
:*
dtype0*
valueB"       29
7tv_text_vectorization/StringSplit/strided_slice/stack_1Ć
7tv_text_vectorization/StringSplit/strided_slice/stack_2Const*
_output_shapes
:*
dtype0*
valueB"      29
7tv_text_vectorization/StringSplit/strided_slice/stack_2ę
/tv_text_vectorization/StringSplit/strided_sliceStridedSlice9tv_text_vectorization/StringSplit/StringSplitV2:indices:0>tv_text_vectorization/StringSplit/strided_slice/stack:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_1:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_2:output:0*
Index0*
T0	*#
_output_shapes
:’’’’’’’’’*

begin_mask*
end_mask*
shrink_axis_mask21
/tv_text_vectorization/StringSplit/strided_slice¼
7tv_text_vectorization/StringSplit/strided_slice_1/stackConst*
_output_shapes
:*
dtype0*
valueB: 29
7tv_text_vectorization/StringSplit/strided_slice_1/stackĄ
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Ą
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2æ
1tv_text_vectorization/StringSplit/strided_slice_1StridedSlice7tv_text_vectorization/StringSplit/StringSplitV2:shape:0@tv_text_vectorization/StringSplit/strided_slice_1/stack:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_1:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_2:output:0*
Index0*
T0	*
_output_shapes
: *
shrink_axis_mask23
1tv_text_vectorization/StringSplit/strided_slice_1³
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CastCast8tv_text_vectorization/StringSplit/strided_slice:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2Z
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast¬
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Cast:tv_text_vectorization/StringSplit/strided_slice_1:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Ō
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ShapeShape\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0*
T0*
_output_shapes
:2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstConst*
_output_shapes
:*
dtype0*
valueB: 2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstÉ
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ProdProdktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const:output:0*
T0*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yConst*
_output_shapes
: *
dtype0*
value	B : 2h
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yÕ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/GreaterGreaterjtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod:output:0otv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y:output:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greaterč
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/CastCasthtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater:z:0*

DstT0*

SrcT0
*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1Const*
_output_shapes
:*
dtype0*
valueB: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaxMax\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yConst*
_output_shapes
: *
dtype0*
value	B :2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yĘ
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/addAddV2itv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mulMuletv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add:z:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul¾
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumMaximum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumĀ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MinimumMinimum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2Const*
_output_shapes
: *
dtype0	*
valueB	 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2æ
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/BincountBincount\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum:z:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2g
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisČ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CumsumCumsumltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount:bins:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0Const*
_output_shapes
:*
dtype0	*
valueB	R 2e
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisµ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatConcatV2ltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0:output:0`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum:out:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis:output:0*
N*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle8tv_text_vectorization/StringSplit/StringSplitV2:values:0Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*#
_output_shapes
:’’’’’’’’’2N
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
5tv_text_vectorization/string_lookup/assert_equal/NoOpNoOp*
_output_shapes
 27
5tv_text_vectorization/string_lookup/assert_equal/NoOpķ
,tv_text_vectorization/string_lookup/IdentityIdentityUtv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
T0	*#
_output_shapes
:’’’’’’’’’2.
,tv_text_vectorization/string_lookup/Identity’
.tv_text_vectorization/string_lookup/Identity_1Identityctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat:output:0*
T0	*#
_output_shapes
:’’’’’’’’’20
.tv_text_vectorization/string_lookup/Identity_1Ŗ
2tv_text_vectorization/RaggedToTensor/default_valueConst*
_output_shapes
: *
dtype0	*
value	B	 R 24
2tv_text_vectorization/RaggedToTensor/default_value£
*tv_text_vectorization/RaggedToTensor/ConstConst*
_output_shapes
: *
dtype0	*
valueB	 R
’’’’’’’’’2,
*tv_text_vectorization/RaggedToTensor/Const
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorRaggedTensorToTensor3tv_text_vectorization/RaggedToTensor/Const:output:05tv_text_vectorization/string_lookup/Identity:output:0;tv_text_vectorization/RaggedToTensor/default_value:output:07tv_text_vectorization/string_lookup/Identity_1:output:0*
T0	*
Tindex0	*
Tshape0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’*
num_row_partition_tensors*%
row_partition_types

ROW_SPLITS2;
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensor
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleinputs_1Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*'
_output_shapes
:’’’’’’’’’2;
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2t
concatenate/concat/axisConst*
_output_shapes
: *
dtype0*
value	B :2
concatenate/concat/axis¢
concatenate/concatConcatV2Btv_text_vectorization/RaggedToTensor/RaggedTensorToTensor:result:0Bsl_string_lookup/None_lookup_table_find/LookupTableFindV2:values:0 concatenate/concat/axis:output:0*
N*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
concatenate/concath
tf.math.add/Add/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.add/Add/y
tf.math.add/AddAddconcatenate/concat:output:0tf.math.add/Add/y:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
tf.math.add/Add
*tf.math.reduce_prod/Prod/reduction_indicesConst*
_output_shapes
: *
dtype0*
value	B :2,
*tf.math.reduce_prod/Prod/reduction_indices“
tf.math.reduce_prod/ProdProdtf.math.add/Add:z:03tf.math.reduce_prod/Prod/reduction_indices:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2
tf.math.reduce_prod/Prodv
tf.math.divide/truediv/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.divide/truediv/y¢
tf.math.divide/truediv/CastCast!tf.math.reduce_prod/Prod:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv/Cast
tf.math.divide/truediv/Cast_1Cast!tf.math.divide/truediv/y:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2
tf.math.divide/truediv/Cast_1­
tf.math.divide/truedivRealDivtf.math.divide/truediv/Cast:y:0!tf.math.divide/truediv/Cast_1:y:0*
T0*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv}
tf.cast/CastCasttf.math.divide/truediv:z:0*

DstT0*

SrcT0*#
_output_shapes
:’’’’’’’’’2
tf.cast/Castp
expand/ExpandDims/dimConst*
_output_shapes
: *
dtype0*
value	B :2
expand/ExpandDims/dim
expand/ExpandDims
ExpandDimstf.cast/Cast:y:0expand/ExpandDims/dim:output:0*
T0*'
_output_shapes
:’’’’’’’’’2
expand/ExpandDimsł
IdentityIdentityexpand/ExpandDims:output:0:^sl_string_lookup/None_lookup_table_find/LookupTableFindV2M^tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 2v
9sl_string_lookup/None_lookup_table_find/LookupTableFindV29sl_string_lookup/None_lookup_table_find/LookupTableFindV22
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:Q M
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/0:QM
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/1:

_output_shapes
: :

_output_shapes
: 
µ
[
?__inference_expand_layer_call_and_return_conditional_losses_962

inputs
identityb
ExpandDims/dimConst*
_output_shapes
: *
dtype0*
value	B :2
ExpandDims/dimy

ExpandDims
ExpandDimsinputsExpandDims/dim:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

ExpandDimsg
IdentityIdentityExpandDims:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*"
_input_shapes
:’’’’’’’’’:K G
#
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs

-
__inference__initializer_1003
identityP
ConstConst*
_output_shapes
: *
dtype0*
value	B :2
ConstQ
IdentityIdentityConst:output:0*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 
Ų
Ņ
L__inference_vectorization_float_layer_call_and_return_conditional_losses_492
tv
sl]
Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle^
Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	J
Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleK
Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	
identity¢9sl_string_lookup/None_lookup_table_find/LookupTableFindV2¢Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
!tv_text_vectorization/StringLowerStringLowertv*'
_output_shapes
:’’’’’’’’’2#
!tv_text_vectorization/StringLower
(tv_text_vectorization/StaticRegexReplaceStaticRegexReplace*tv_text_vectorization/StringLower:output:0*'
_output_shapes
:’’’’’’’’’*6
pattern+)[!"#$%&()\*\+,-\./:;<=>?@\[\\\]^_`{|}~\']*
rewrite 2*
(tv_text_vectorization/StaticRegexReplaceŹ
tv_text_vectorization/SqueezeSqueeze1tv_text_vectorization/StaticRegexReplace:output:0*
T0*#
_output_shapes
:’’’’’’’’’*
squeeze_dims

’’’’’’’’’2
tv_text_vectorization/Squeeze
'tv_text_vectorization/StringSplit/ConstConst*
_output_shapes
: *
dtype0*
valueB B 2)
'tv_text_vectorization/StringSplit/Const
/tv_text_vectorization/StringSplit/StringSplitV2StringSplitV2&tv_text_vectorization/Squeeze:output:00tv_text_vectorization/StringSplit/Const:output:0*<
_output_shapes*
(:’’’’’’’’’:’’’’’’’’’:21
/tv_text_vectorization/StringSplit/StringSplitV2æ
5tv_text_vectorization/StringSplit/strided_slice/stackConst*
_output_shapes
:*
dtype0*
valueB"        27
5tv_text_vectorization/StringSplit/strided_slice/stackĆ
7tv_text_vectorization/StringSplit/strided_slice/stack_1Const*
_output_shapes
:*
dtype0*
valueB"       29
7tv_text_vectorization/StringSplit/strided_slice/stack_1Ć
7tv_text_vectorization/StringSplit/strided_slice/stack_2Const*
_output_shapes
:*
dtype0*
valueB"      29
7tv_text_vectorization/StringSplit/strided_slice/stack_2ę
/tv_text_vectorization/StringSplit/strided_sliceStridedSlice9tv_text_vectorization/StringSplit/StringSplitV2:indices:0>tv_text_vectorization/StringSplit/strided_slice/stack:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_1:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_2:output:0*
Index0*
T0	*#
_output_shapes
:’’’’’’’’’*

begin_mask*
end_mask*
shrink_axis_mask21
/tv_text_vectorization/StringSplit/strided_slice¼
7tv_text_vectorization/StringSplit/strided_slice_1/stackConst*
_output_shapes
:*
dtype0*
valueB: 29
7tv_text_vectorization/StringSplit/strided_slice_1/stackĄ
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Ą
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2æ
1tv_text_vectorization/StringSplit/strided_slice_1StridedSlice7tv_text_vectorization/StringSplit/StringSplitV2:shape:0@tv_text_vectorization/StringSplit/strided_slice_1/stack:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_1:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_2:output:0*
Index0*
T0	*
_output_shapes
: *
shrink_axis_mask23
1tv_text_vectorization/StringSplit/strided_slice_1³
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CastCast8tv_text_vectorization/StringSplit/strided_slice:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2Z
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast¬
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Cast:tv_text_vectorization/StringSplit/strided_slice_1:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Ō
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ShapeShape\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0*
T0*
_output_shapes
:2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstConst*
_output_shapes
:*
dtype0*
valueB: 2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstÉ
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ProdProdktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const:output:0*
T0*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yConst*
_output_shapes
: *
dtype0*
value	B : 2h
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yÕ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/GreaterGreaterjtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod:output:0otv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y:output:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greaterč
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/CastCasthtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater:z:0*

DstT0*

SrcT0
*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1Const*
_output_shapes
:*
dtype0*
valueB: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaxMax\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yConst*
_output_shapes
: *
dtype0*
value	B :2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yĘ
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/addAddV2itv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mulMuletv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add:z:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul¾
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumMaximum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumĀ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MinimumMinimum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2Const*
_output_shapes
: *
dtype0	*
valueB	 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2æ
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/BincountBincount\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum:z:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2g
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisČ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CumsumCumsumltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount:bins:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0Const*
_output_shapes
:*
dtype0	*
valueB	R 2e
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisµ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatConcatV2ltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0:output:0`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum:out:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis:output:0*
N*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle8tv_text_vectorization/StringSplit/StringSplitV2:values:0Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*#
_output_shapes
:’’’’’’’’’2N
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
5tv_text_vectorization/string_lookup/assert_equal/NoOpNoOp*
_output_shapes
 27
5tv_text_vectorization/string_lookup/assert_equal/NoOpķ
,tv_text_vectorization/string_lookup/IdentityIdentityUtv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
T0	*#
_output_shapes
:’’’’’’’’’2.
,tv_text_vectorization/string_lookup/Identity’
.tv_text_vectorization/string_lookup/Identity_1Identityctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat:output:0*
T0	*#
_output_shapes
:’’’’’’’’’20
.tv_text_vectorization/string_lookup/Identity_1Ŗ
2tv_text_vectorization/RaggedToTensor/default_valueConst*
_output_shapes
: *
dtype0	*
value	B	 R 24
2tv_text_vectorization/RaggedToTensor/default_value£
*tv_text_vectorization/RaggedToTensor/ConstConst*
_output_shapes
: *
dtype0	*
valueB	 R
’’’’’’’’’2,
*tv_text_vectorization/RaggedToTensor/Const
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorRaggedTensorToTensor3tv_text_vectorization/RaggedToTensor/Const:output:05tv_text_vectorization/string_lookup/Identity:output:0;tv_text_vectorization/RaggedToTensor/default_value:output:07tv_text_vectorization/string_lookup/Identity_1:output:0*
T0	*
Tindex0	*
Tshape0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’*
num_row_partition_tensors*%
row_partition_types

ROW_SPLITS2;
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensor
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleslGsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*'
_output_shapes
:’’’’’’’’’2;
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2ä
concatenate/PartitionedCallPartitionedCallBtv_text_vectorization/RaggedToTensor/RaggedTensorToTensor:result:0Bsl_string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
Tin
2		*
Tout
2	*
_collective_manager_ids
 *0
_output_shapes
:’’’’’’’’’’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *M
fHRF
D__inference_concatenate_layer_call_and_return_conditional_losses_4482
concatenate/PartitionedCallh
tf.math.add/Add/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.add/Add/y¦
tf.math.add/AddAdd$concatenate/PartitionedCall:output:0tf.math.add/Add/y:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
tf.math.add/Add
*tf.math.reduce_prod/Prod/reduction_indicesConst*
_output_shapes
: *
dtype0*
value	B :2,
*tf.math.reduce_prod/Prod/reduction_indices“
tf.math.reduce_prod/ProdProdtf.math.add/Add:z:03tf.math.reduce_prod/Prod/reduction_indices:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2
tf.math.reduce_prod/Prodv
tf.math.divide/truediv/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.divide/truediv/y¢
tf.math.divide/truediv/CastCast!tf.math.reduce_prod/Prod:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv/Cast
tf.math.divide/truediv/Cast_1Cast!tf.math.divide/truediv/y:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2
tf.math.divide/truediv/Cast_1­
tf.math.divide/truedivRealDivtf.math.divide/truediv/Cast:y:0!tf.math.divide/truediv/Cast_1:y:0*
T0*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv}
tf.cast/CastCasttf.math.divide/truediv:z:0*

DstT0*

SrcT0*#
_output_shapes
:’’’’’’’’’2
tf.cast/CastÕ
expand/PartitionedCallPartitionedCalltf.cast/Cast:y:0*
Tin
2*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *H
fCRA
?__inference_expand_layer_call_and_return_conditional_losses_4722
expand/PartitionedCallž
IdentityIdentityexpand/PartitionedCall:output:0:^sl_string_lookup/None_lookup_table_find/LookupTableFindV2M^tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 2v
9sl_string_lookup/None_lookup_table_find/LookupTableFindV29sl_string_lookup/None_lookup_table_find/LookupTableFindV22
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:K G
'
_output_shapes
:’’’’’’’’’

_user_specified_nametv:KG
'
_output_shapes
:’’’’’’’’’

_user_specified_namesl:

_output_shapes
: :

_output_shapes
: 

+
__inference__destroyer_1008
identityP
ConstConst*
_output_shapes
: *
dtype0*
value	B :2
ConstQ
IdentityIdentityConst:output:0*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 
ģ
»
__inference_save_fn_1027
checkpoint_key\
Xsl_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2_table_handle
identity

identity_1

identity_2

identity_3

identity_4

identity_5	¢Ksl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2ó
Ksl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2LookupTableExportV2Xsl_string_lookup_index_table_lookup_table_export_values_lookuptableexportv2_table_handle",/job:localhost/replica:0/task:0/device:CPU:0*
Tkeys0*
Tvalues0	*
_output_shapes

::2M
Ksl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2T
add/yConst*
_output_shapes
: *
dtype0*
valueB B-keys2
add/yR
addAddcheckpoint_keyadd/y:output:0*
T0*
_output_shapes
: 2
addZ
add_1/yConst*
_output_shapes
: *
dtype0*
valueB B-values2	
add_1/yX
add_1Addcheckpoint_keyadd_1/y:output:0*
T0*
_output_shapes
: 2
add_1
IdentityIdentityadd:z:0L^sl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

IdentityO
ConstConst*
_output_shapes
: *
dtype0*
valueB B 2
Const£

Identity_1IdentityConst:output:0L^sl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

Identity_1é

Identity_2IdentityRsl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2:keys:0L^sl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
:2

Identity_2

Identity_3Identity	add_1:z:0L^sl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

Identity_3S
Const_1Const*
_output_shapes
: *
dtype0*
valueB B 2	
Const_1„

Identity_4IdentityConst_1:output:0L^sl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

Identity_4ė

Identity_5IdentityTsl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2:values:0L^sl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0	*
_output_shapes
:2

Identity_5"
identityIdentity:output:0"!

identity_1Identity_1:output:0"!

identity_2Identity_2:output:0"!

identity_3Identity_3:output:0"!

identity_4Identity_4:output:0"!

identity_5Identity_5:output:0*
_input_shapes
: :2
Ksl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2Ksl_string_lookup_index_table_lookup_table_export_values/LookupTableExportV2:F B

_output_shapes
: 
(
_user_specified_namecheckpoint_key
æ
µ
__inference_save_fn_1054
checkpoint_keyY
Ustring_lookup_index_table_lookup_table_export_values_lookuptableexportv2_table_handle
identity

identity_1

identity_2

identity_3

identity_4

identity_5	¢Hstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2ź
Hstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2LookupTableExportV2Ustring_lookup_index_table_lookup_table_export_values_lookuptableexportv2_table_handle",/job:localhost/replica:0/task:0/device:CPU:0*
Tkeys0*
Tvalues0	*
_output_shapes

::2J
Hstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2T
add/yConst*
_output_shapes
: *
dtype0*
valueB B-keys2
add/yR
addAddcheckpoint_keyadd/y:output:0*
T0*
_output_shapes
: 2
addZ
add_1/yConst*
_output_shapes
: *
dtype0*
valueB B-values2	
add_1/yX
add_1Addcheckpoint_keyadd_1/y:output:0*
T0*
_output_shapes
: 2
add_1
IdentityIdentityadd:z:0I^string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

IdentityO
ConstConst*
_output_shapes
: *
dtype0*
valueB B 2
Const 

Identity_1IdentityConst:output:0I^string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

Identity_1ć

Identity_2IdentityOstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2:keys:0I^string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
:2

Identity_2

Identity_3Identity	add_1:z:0I^string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

Identity_3S
Const_1Const*
_output_shapes
: *
dtype0*
valueB B 2	
Const_1¢

Identity_4IdentityConst_1:output:0I^string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0*
_output_shapes
: 2

Identity_4å

Identity_5IdentityQstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2:values:0I^string_lookup_index_table_lookup_table_export_values/LookupTableExportV2*
T0	*
_output_shapes
:2

Identity_5"
identityIdentity:output:0"!

identity_1Identity_1:output:0"!

identity_2Identity_2:output:0"!

identity_3Identity_3:output:0"!

identity_4Identity_4:output:0"!

identity_5Identity_5:output:0*
_input_shapes
: :2
Hstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2Hstring_lookup_index_table_lookup_table_export_values/LookupTableExportV2:F B

_output_shapes
: 
(
_user_specified_namecheckpoint_key
ō
Ü
L__inference_vectorization_float_layer_call_and_return_conditional_losses_702

inputs
inputs_1]
Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle^
Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	J
Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleK
Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	
identity¢9sl_string_lookup/None_lookup_table_find/LookupTableFindV2¢Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
!tv_text_vectorization/StringLowerStringLowerinputs*'
_output_shapes
:’’’’’’’’’2#
!tv_text_vectorization/StringLower
(tv_text_vectorization/StaticRegexReplaceStaticRegexReplace*tv_text_vectorization/StringLower:output:0*'
_output_shapes
:’’’’’’’’’*6
pattern+)[!"#$%&()\*\+,-\./:;<=>?@\[\\\]^_`{|}~\']*
rewrite 2*
(tv_text_vectorization/StaticRegexReplaceŹ
tv_text_vectorization/SqueezeSqueeze1tv_text_vectorization/StaticRegexReplace:output:0*
T0*#
_output_shapes
:’’’’’’’’’*
squeeze_dims

’’’’’’’’’2
tv_text_vectorization/Squeeze
'tv_text_vectorization/StringSplit/ConstConst*
_output_shapes
: *
dtype0*
valueB B 2)
'tv_text_vectorization/StringSplit/Const
/tv_text_vectorization/StringSplit/StringSplitV2StringSplitV2&tv_text_vectorization/Squeeze:output:00tv_text_vectorization/StringSplit/Const:output:0*<
_output_shapes*
(:’’’’’’’’’:’’’’’’’’’:21
/tv_text_vectorization/StringSplit/StringSplitV2æ
5tv_text_vectorization/StringSplit/strided_slice/stackConst*
_output_shapes
:*
dtype0*
valueB"        27
5tv_text_vectorization/StringSplit/strided_slice/stackĆ
7tv_text_vectorization/StringSplit/strided_slice/stack_1Const*
_output_shapes
:*
dtype0*
valueB"       29
7tv_text_vectorization/StringSplit/strided_slice/stack_1Ć
7tv_text_vectorization/StringSplit/strided_slice/stack_2Const*
_output_shapes
:*
dtype0*
valueB"      29
7tv_text_vectorization/StringSplit/strided_slice/stack_2ę
/tv_text_vectorization/StringSplit/strided_sliceStridedSlice9tv_text_vectorization/StringSplit/StringSplitV2:indices:0>tv_text_vectorization/StringSplit/strided_slice/stack:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_1:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_2:output:0*
Index0*
T0	*#
_output_shapes
:’’’’’’’’’*

begin_mask*
end_mask*
shrink_axis_mask21
/tv_text_vectorization/StringSplit/strided_slice¼
7tv_text_vectorization/StringSplit/strided_slice_1/stackConst*
_output_shapes
:*
dtype0*
valueB: 29
7tv_text_vectorization/StringSplit/strided_slice_1/stackĄ
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Ą
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2æ
1tv_text_vectorization/StringSplit/strided_slice_1StridedSlice7tv_text_vectorization/StringSplit/StringSplitV2:shape:0@tv_text_vectorization/StringSplit/strided_slice_1/stack:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_1:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_2:output:0*
Index0*
T0	*
_output_shapes
: *
shrink_axis_mask23
1tv_text_vectorization/StringSplit/strided_slice_1³
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CastCast8tv_text_vectorization/StringSplit/strided_slice:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2Z
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast¬
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Cast:tv_text_vectorization/StringSplit/strided_slice_1:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Ō
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ShapeShape\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0*
T0*
_output_shapes
:2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstConst*
_output_shapes
:*
dtype0*
valueB: 2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstÉ
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ProdProdktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const:output:0*
T0*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yConst*
_output_shapes
: *
dtype0*
value	B : 2h
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yÕ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/GreaterGreaterjtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod:output:0otv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y:output:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greaterč
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/CastCasthtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater:z:0*

DstT0*

SrcT0
*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1Const*
_output_shapes
:*
dtype0*
valueB: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaxMax\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yConst*
_output_shapes
: *
dtype0*
value	B :2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yĘ
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/addAddV2itv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mulMuletv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add:z:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul¾
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumMaximum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumĀ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MinimumMinimum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2Const*
_output_shapes
: *
dtype0	*
valueB	 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2æ
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/BincountBincount\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum:z:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2g
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisČ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CumsumCumsumltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount:bins:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0Const*
_output_shapes
:*
dtype0	*
valueB	R 2e
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisµ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatConcatV2ltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0:output:0`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum:out:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis:output:0*
N*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle8tv_text_vectorization/StringSplit/StringSplitV2:values:0Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*#
_output_shapes
:’’’’’’’’’2N
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
5tv_text_vectorization/string_lookup/assert_equal/NoOpNoOp*
_output_shapes
 27
5tv_text_vectorization/string_lookup/assert_equal/NoOpķ
,tv_text_vectorization/string_lookup/IdentityIdentityUtv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
T0	*#
_output_shapes
:’’’’’’’’’2.
,tv_text_vectorization/string_lookup/Identity’
.tv_text_vectorization/string_lookup/Identity_1Identityctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat:output:0*
T0	*#
_output_shapes
:’’’’’’’’’20
.tv_text_vectorization/string_lookup/Identity_1Ŗ
2tv_text_vectorization/RaggedToTensor/default_valueConst*
_output_shapes
: *
dtype0	*
value	B	 R 24
2tv_text_vectorization/RaggedToTensor/default_value£
*tv_text_vectorization/RaggedToTensor/ConstConst*
_output_shapes
: *
dtype0	*
valueB	 R
’’’’’’’’’2,
*tv_text_vectorization/RaggedToTensor/Const
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorRaggedTensorToTensor3tv_text_vectorization/RaggedToTensor/Const:output:05tv_text_vectorization/string_lookup/Identity:output:0;tv_text_vectorization/RaggedToTensor/default_value:output:07tv_text_vectorization/string_lookup/Identity_1:output:0*
T0	*
Tindex0	*
Tshape0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’*
num_row_partition_tensors*%
row_partition_types

ROW_SPLITS2;
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensor
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleinputs_1Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*'
_output_shapes
:’’’’’’’’’2;
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2ä
concatenate/PartitionedCallPartitionedCallBtv_text_vectorization/RaggedToTensor/RaggedTensorToTensor:result:0Bsl_string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
Tin
2		*
Tout
2	*
_collective_manager_ids
 *0
_output_shapes
:’’’’’’’’’’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *M
fHRF
D__inference_concatenate_layer_call_and_return_conditional_losses_4482
concatenate/PartitionedCallh
tf.math.add/Add/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.add/Add/y¦
tf.math.add/AddAdd$concatenate/PartitionedCall:output:0tf.math.add/Add/y:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
tf.math.add/Add
*tf.math.reduce_prod/Prod/reduction_indicesConst*
_output_shapes
: *
dtype0*
value	B :2,
*tf.math.reduce_prod/Prod/reduction_indices“
tf.math.reduce_prod/ProdProdtf.math.add/Add:z:03tf.math.reduce_prod/Prod/reduction_indices:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2
tf.math.reduce_prod/Prodv
tf.math.divide/truediv/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.divide/truediv/y¢
tf.math.divide/truediv/CastCast!tf.math.reduce_prod/Prod:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv/Cast
tf.math.divide/truediv/Cast_1Cast!tf.math.divide/truediv/y:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2
tf.math.divide/truediv/Cast_1­
tf.math.divide/truedivRealDivtf.math.divide/truediv/Cast:y:0!tf.math.divide/truediv/Cast_1:y:0*
T0*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv}
tf.cast/CastCasttf.math.divide/truediv:z:0*

DstT0*

SrcT0*#
_output_shapes
:’’’’’’’’’2
tf.cast/CastÕ
expand/PartitionedCallPartitionedCalltf.cast/Cast:y:0*
Tin
2*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *H
fCRA
?__inference_expand_layer_call_and_return_conditional_losses_4782
expand/PartitionedCallž
IdentityIdentityexpand/PartitionedCall:output:0:^sl_string_lookup/None_lookup_table_find/LookupTableFindV2M^tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 2v
9sl_string_lookup/None_lookup_table_find/LookupTableFindV29sl_string_lookup/None_lookup_table_find/LookupTableFindV22
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:O K
'
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs:OK
'
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs:

_output_shapes
: :

_output_shapes
: 
Õ
n
D__inference_concatenate_layer_call_and_return_conditional_losses_448

inputs	
inputs_1	
identity	\
concat/axisConst*
_output_shapes
: *
dtype0*
value	B :2
concat/axis
concatConcatV2inputsinputs_1concat/axis:output:0*
N*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
concatl
IdentityIdentityconcat:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2

Identity"
identityIdentity:output:0*B
_input_shapes1
/:’’’’’’’’’’’’’’’’’’:’’’’’’’’’:X T
0
_output_shapes
:’’’’’’’’’’’’’’’’’’
 
_user_specified_nameinputs:OK
'
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs
ų
J
__inference__creator_983
identity¢sl_string_lookup_index_table©
sl_string_lookup_index_tableMutableHashTableV2*
_output_shapes
: *
	key_dtype0*
shared_name
table_73*
value_dtype0	2
sl_string_lookup_index_table
IdentityIdentity+sl_string_lookup_index_table:table_handle:0^sl_string_lookup_index_table*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 2<
sl_string_lookup_index_tablesl_string_lookup_index_table
Ą
Ø
1__inference_vectorization_float_layer_call_fn_635
tv
sl
unknown
	unknown_0	
	unknown_1
	unknown_2	
identity¢StatefulPartitionedCall
StatefulPartitionedCallStatefulPartitionedCalltvslunknown	unknown_0	unknown_1	unknown_2*
Tin

2		*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *U
fPRN
L__inference_vectorization_float_layer_call_and_return_conditional_losses_6242
StatefulPartitionedCall
IdentityIdentity StatefulPartitionedCall:output:0^StatefulPartitionedCall*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 22
StatefulPartitionedCallStatefulPartitionedCall:K G
'
_output_shapes
:’’’’’’’’’

_user_specified_nametv:KG
'
_output_shapes
:’’’’’’’’’

_user_specified_namesl:

_output_shapes
: :

_output_shapes
: 
ä
“
1__inference_vectorization_float_layer_call_fn_929
inputs_0
inputs_1
unknown
	unknown_0	
	unknown_1
	unknown_2	
identity¢StatefulPartitionedCall
StatefulPartitionedCallStatefulPartitionedCallinputs_0inputs_1unknown	unknown_0	unknown_1	unknown_2*
Tin

2		*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *U
fPRN
L__inference_vectorization_float_layer_call_and_return_conditional_losses_6242
StatefulPartitionedCall
IdentityIdentity StatefulPartitionedCall:output:0^StatefulPartitionedCall*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 22
StatefulPartitionedCallStatefulPartitionedCall:Q M
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/0:QM
'
_output_shapes
:’’’’’’’’’
"
_user_specified_name
inputs/1:

_output_shapes
: :

_output_shapes
: 
Ą¤

__inference__wrapped_model_388
tv
slq
mvectorization_float_tv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handler
nvectorization_float_tv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	^
Zvectorization_float_sl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle_
[vectorization_float_sl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	
identity¢Mvectorization_float/sl_string_lookup/None_lookup_table_find/LookupTableFindV2¢`vectorization_float/tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2Ŗ
5vectorization_float/tv_text_vectorization/StringLowerStringLowertv*'
_output_shapes
:’’’’’’’’’27
5vectorization_float/tv_text_vectorization/StringLowerĀ
<vectorization_float/tv_text_vectorization/StaticRegexReplaceStaticRegexReplace>vectorization_float/tv_text_vectorization/StringLower:output:0*'
_output_shapes
:’’’’’’’’’*6
pattern+)[!"#$%&()\*\+,-\./:;<=>?@\[\\\]^_`{|}~\']*
rewrite 2>
<vectorization_float/tv_text_vectorization/StaticRegexReplace
1vectorization_float/tv_text_vectorization/SqueezeSqueezeEvectorization_float/tv_text_vectorization/StaticRegexReplace:output:0*
T0*#
_output_shapes
:’’’’’’’’’*
squeeze_dims

’’’’’’’’’23
1vectorization_float/tv_text_vectorization/Squeeze»
;vectorization_float/tv_text_vectorization/StringSplit/ConstConst*
_output_shapes
: *
dtype0*
valueB B 2=
;vectorization_float/tv_text_vectorization/StringSplit/ConstŪ
Cvectorization_float/tv_text_vectorization/StringSplit/StringSplitV2StringSplitV2:vectorization_float/tv_text_vectorization/Squeeze:output:0Dvectorization_float/tv_text_vectorization/StringSplit/Const:output:0*<
_output_shapes*
(:’’’’’’’’’:’’’’’’’’’:2E
Cvectorization_float/tv_text_vectorization/StringSplit/StringSplitV2ē
Ivectorization_float/tv_text_vectorization/StringSplit/strided_slice/stackConst*
_output_shapes
:*
dtype0*
valueB"        2K
Ivectorization_float/tv_text_vectorization/StringSplit/strided_slice/stackė
Kvectorization_float/tv_text_vectorization/StringSplit/strided_slice/stack_1Const*
_output_shapes
:*
dtype0*
valueB"       2M
Kvectorization_float/tv_text_vectorization/StringSplit/strided_slice/stack_1ė
Kvectorization_float/tv_text_vectorization/StringSplit/strided_slice/stack_2Const*
_output_shapes
:*
dtype0*
valueB"      2M
Kvectorization_float/tv_text_vectorization/StringSplit/strided_slice/stack_2Ž
Cvectorization_float/tv_text_vectorization/StringSplit/strided_sliceStridedSliceMvectorization_float/tv_text_vectorization/StringSplit/StringSplitV2:indices:0Rvectorization_float/tv_text_vectorization/StringSplit/strided_slice/stack:output:0Tvectorization_float/tv_text_vectorization/StringSplit/strided_slice/stack_1:output:0Tvectorization_float/tv_text_vectorization/StringSplit/strided_slice/stack_2:output:0*
Index0*
T0	*#
_output_shapes
:’’’’’’’’’*

begin_mask*
end_mask*
shrink_axis_mask2E
Cvectorization_float/tv_text_vectorization/StringSplit/strided_sliceä
Kvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stackConst*
_output_shapes
:*
dtype0*
valueB: 2M
Kvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stackč
Mvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stack_1Const*
_output_shapes
:*
dtype0*
valueB:2O
Mvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stack_1č
Mvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stack_2Const*
_output_shapes
:*
dtype0*
valueB:2O
Mvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stack_2·
Evectorization_float/tv_text_vectorization/StringSplit/strided_slice_1StridedSliceKvectorization_float/tv_text_vectorization/StringSplit/StringSplitV2:shape:0Tvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stack:output:0Vvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stack_1:output:0Vvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1/stack_2:output:0*
Index0*
T0	*
_output_shapes
: *
shrink_axis_mask2G
Evectorization_float/tv_text_vectorization/StringSplit/strided_slice_1ļ
lvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CastCastLvectorization_float/tv_text_vectorization/StringSplit/strided_slice:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2n
lvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Castč
nvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1CastNvectorization_float/tv_text_vectorization/StringSplit/strided_slice_1:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2p
nvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1
vvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ShapeShapepvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0*
T0*
_output_shapes
:2x
vvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shapeŗ
vvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstConst*
_output_shapes
:*
dtype0*
valueB: 2x
vvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const
uvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ProdProdvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape:output:0vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const:output:0*
T0*
_output_shapes
: 2w
uvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prodŗ
zvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yConst*
_output_shapes
: *
dtype0*
value	B : 2|
zvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y¦
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/GreaterGreater~vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod:output:0vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y:output:0*
T0*
_output_shapes
: 2z
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater¤
uvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/CastCast|vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater:z:0*

DstT0*

SrcT0
*
_output_shapes
: 2w
uvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast¾
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1Const*
_output_shapes
:*
dtype0*
valueB: 2z
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1
tvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaxMaxpvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1:output:0*
T0*
_output_shapes
: 2v
tvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max²
vvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yConst*
_output_shapes
: *
dtype0*
value	B :2x
vvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y
tvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/addAddV2}vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max:output:0vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y:output:0*
T0*
_output_shapes
: 2v
tvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add
tvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mulMulyvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast:y:0xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add:z:0*
T0*
_output_shapes
: 2v
tvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumMaximumrvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul:z:0*
T0*
_output_shapes
: 2z
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MinimumMinimumrvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0|vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum:z:0*
T0*
_output_shapes
: 2z
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum·
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2Const*
_output_shapes
: *
dtype0	*
valueB	 2z
xvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2¤
yvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/BincountBincountpvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0|vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum:z:0vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2{
yvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount¬
svectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisConst*
_output_shapes
: *
dtype0*
value	B : 2u
svectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis
nvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CumsumCumsumvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount:bins:0|vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2p
nvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum¼
wvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0Const*
_output_shapes
:*
dtype0	*
valueB	R 2y
wvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0¬
svectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisConst*
_output_shapes
: *
dtype0*
value	B : 2u
svectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis
nvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatConcatV2vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0:output:0tvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum:out:0|vectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis:output:0*
N*
T0	*#
_output_shapes
:’’’’’’’’’2p
nvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatš
`vectorization_float/tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2mvectorization_float_tv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleLvectorization_float/tv_text_vectorization/StringSplit/StringSplitV2:values:0nvectorization_float_tv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*#
_output_shapes
:’’’’’’’’’2b
`vectorization_float/tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2“
Ivectorization_float/tv_text_vectorization/string_lookup/assert_equal/NoOpNoOp*
_output_shapes
 2K
Ivectorization_float/tv_text_vectorization/string_lookup/assert_equal/NoOp©
@vectorization_float/tv_text_vectorization/string_lookup/IdentityIdentityivectorization_float/tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
T0	*#
_output_shapes
:’’’’’’’’’2B
@vectorization_float/tv_text_vectorization/string_lookup/Identity»
Bvectorization_float/tv_text_vectorization/string_lookup/Identity_1Identitywvectorization_float/tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2D
Bvectorization_float/tv_text_vectorization/string_lookup/Identity_1Ņ
Fvectorization_float/tv_text_vectorization/RaggedToTensor/default_valueConst*
_output_shapes
: *
dtype0	*
value	B	 R 2H
Fvectorization_float/tv_text_vectorization/RaggedToTensor/default_valueĖ
>vectorization_float/tv_text_vectorization/RaggedToTensor/ConstConst*
_output_shapes
: *
dtype0	*
valueB	 R
’’’’’’’’’2@
>vectorization_float/tv_text_vectorization/RaggedToTensor/Const
Mvectorization_float/tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorRaggedTensorToTensorGvectorization_float/tv_text_vectorization/RaggedToTensor/Const:output:0Ivectorization_float/tv_text_vectorization/string_lookup/Identity:output:0Ovectorization_float/tv_text_vectorization/RaggedToTensor/default_value:output:0Kvectorization_float/tv_text_vectorization/string_lookup/Identity_1:output:0*
T0	*
Tindex0	*
Tshape0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’*
num_row_partition_tensors*%
row_partition_types

ROW_SPLITS2O
Mvectorization_float/tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorŽ
Mvectorization_float/sl_string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Zvectorization_float_sl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handlesl[vectorization_float_sl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*'
_output_shapes
:’’’’’’’’’2O
Mvectorization_float/sl_string_lookup/None_lookup_table_find/LookupTableFindV2
+vectorization_float/concatenate/concat/axisConst*
_output_shapes
: *
dtype0*
value	B :2-
+vectorization_float/concatenate/concat/axis
&vectorization_float/concatenate/concatConcatV2Vvectorization_float/tv_text_vectorization/RaggedToTensor/RaggedTensorToTensor:result:0Vvectorization_float/sl_string_lookup/None_lookup_table_find/LookupTableFindV2:values:04vectorization_float/concatenate/concat/axis:output:0*
N*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2(
&vectorization_float/concatenate/concat
%vectorization_float/tf.math.add/Add/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2'
%vectorization_float/tf.math.add/Add/yķ
#vectorization_float/tf.math.add/AddAdd/vectorization_float/concatenate/concat:output:0.vectorization_float/tf.math.add/Add/y:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2%
#vectorization_float/tf.math.add/AddĀ
>vectorization_float/tf.math.reduce_prod/Prod/reduction_indicesConst*
_output_shapes
: *
dtype0*
value	B :2@
>vectorization_float/tf.math.reduce_prod/Prod/reduction_indices
,vectorization_float/tf.math.reduce_prod/ProdProd'vectorization_float/tf.math.add/Add:z:0Gvectorization_float/tf.math.reduce_prod/Prod/reduction_indices:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2.
,vectorization_float/tf.math.reduce_prod/Prod
,vectorization_float/tf.math.divide/truediv/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2.
,vectorization_float/tf.math.divide/truediv/yŽ
/vectorization_float/tf.math.divide/truediv/CastCast5vectorization_float/tf.math.reduce_prod/Prod:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’21
/vectorization_float/tf.math.divide/truediv/CastÕ
1vectorization_float/tf.math.divide/truediv/Cast_1Cast5vectorization_float/tf.math.divide/truediv/y:output:0*

DstT0*

SrcT0	*
_output_shapes
: 23
1vectorization_float/tf.math.divide/truediv/Cast_1ż
*vectorization_float/tf.math.divide/truedivRealDiv3vectorization_float/tf.math.divide/truediv/Cast:y:05vectorization_float/tf.math.divide/truediv/Cast_1:y:0*
T0*#
_output_shapes
:’’’’’’’’’2,
*vectorization_float/tf.math.divide/truediv¹
 vectorization_float/tf.cast/CastCast.vectorization_float/tf.math.divide/truediv:z:0*

DstT0*

SrcT0*#
_output_shapes
:’’’’’’’’’2"
 vectorization_float/tf.cast/Cast
)vectorization_float/expand/ExpandDims/dimConst*
_output_shapes
: *
dtype0*
value	B :2+
)vectorization_float/expand/ExpandDims/dimč
%vectorization_float/expand/ExpandDims
ExpandDims$vectorization_float/tf.cast/Cast:y:02vectorization_float/expand/ExpandDims/dim:output:0*
T0*'
_output_shapes
:’’’’’’’’’2'
%vectorization_float/expand/ExpandDimsµ
IdentityIdentity.vectorization_float/expand/ExpandDims:output:0N^vectorization_float/sl_string_lookup/None_lookup_table_find/LookupTableFindV2a^vectorization_float/tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 2
Mvectorization_float/sl_string_lookup/None_lookup_table_find/LookupTableFindV2Mvectorization_float/sl_string_lookup/None_lookup_table_find/LookupTableFindV22Ä
`vectorization_float/tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2`vectorization_float/tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:K G
'
_output_shapes
:’’’’’’’’’

_user_specified_nametv:KG
'
_output_shapes
:’’’’’’’’’

_user_specified_namesl:

_output_shapes
: :

_output_shapes
: 
£	
š
__inference_restore_fn_1035
restored_tensors_0
restored_tensors_1	O
Ksl_string_lookup_index_table_table_restore_lookuptableimportv2_table_handle
identity¢>sl_string_lookup_index_table_table_restore/LookupTableImportV2ē
>sl_string_lookup_index_table_table_restore/LookupTableImportV2LookupTableImportV2Ksl_string_lookup_index_table_table_restore_lookuptableimportv2_table_handlerestored_tensors_0restored_tensors_1",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*
_output_shapes
 2@
>sl_string_lookup_index_table_table_restore/LookupTableImportV2P
ConstConst*
_output_shapes
: *
dtype0*
value	B :2
Const
IdentityIdentityConst:output:0?^sl_string_lookup_index_table_table_restore/LookupTableImportV2*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes
:::2
>sl_string_lookup_index_table_table_restore/LookupTableImportV2>sl_string_lookup_index_table_table_restore/LookupTableImportV2:L H

_output_shapes
:
,
_user_specified_namerestored_tensors_0:LH

_output_shapes
:
,
_user_specified_namerestored_tensors_1
Ų
Ņ
L__inference_vectorization_float_layer_call_and_return_conditional_losses_556
tv
sl]
Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle^
Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	J
Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleK
Gsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value	
identity¢9sl_string_lookup/None_lookup_table_find/LookupTableFindV2¢Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
!tv_text_vectorization/StringLowerStringLowertv*'
_output_shapes
:’’’’’’’’’2#
!tv_text_vectorization/StringLower
(tv_text_vectorization/StaticRegexReplaceStaticRegexReplace*tv_text_vectorization/StringLower:output:0*'
_output_shapes
:’’’’’’’’’*6
pattern+)[!"#$%&()\*\+,-\./:;<=>?@\[\\\]^_`{|}~\']*
rewrite 2*
(tv_text_vectorization/StaticRegexReplaceŹ
tv_text_vectorization/SqueezeSqueeze1tv_text_vectorization/StaticRegexReplace:output:0*
T0*#
_output_shapes
:’’’’’’’’’*
squeeze_dims

’’’’’’’’’2
tv_text_vectorization/Squeeze
'tv_text_vectorization/StringSplit/ConstConst*
_output_shapes
: *
dtype0*
valueB B 2)
'tv_text_vectorization/StringSplit/Const
/tv_text_vectorization/StringSplit/StringSplitV2StringSplitV2&tv_text_vectorization/Squeeze:output:00tv_text_vectorization/StringSplit/Const:output:0*<
_output_shapes*
(:’’’’’’’’’:’’’’’’’’’:21
/tv_text_vectorization/StringSplit/StringSplitV2æ
5tv_text_vectorization/StringSplit/strided_slice/stackConst*
_output_shapes
:*
dtype0*
valueB"        27
5tv_text_vectorization/StringSplit/strided_slice/stackĆ
7tv_text_vectorization/StringSplit/strided_slice/stack_1Const*
_output_shapes
:*
dtype0*
valueB"       29
7tv_text_vectorization/StringSplit/strided_slice/stack_1Ć
7tv_text_vectorization/StringSplit/strided_slice/stack_2Const*
_output_shapes
:*
dtype0*
valueB"      29
7tv_text_vectorization/StringSplit/strided_slice/stack_2ę
/tv_text_vectorization/StringSplit/strided_sliceStridedSlice9tv_text_vectorization/StringSplit/StringSplitV2:indices:0>tv_text_vectorization/StringSplit/strided_slice/stack:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_1:output:0@tv_text_vectorization/StringSplit/strided_slice/stack_2:output:0*
Index0*
T0	*#
_output_shapes
:’’’’’’’’’*

begin_mask*
end_mask*
shrink_axis_mask21
/tv_text_vectorization/StringSplit/strided_slice¼
7tv_text_vectorization/StringSplit/strided_slice_1/stackConst*
_output_shapes
:*
dtype0*
valueB: 29
7tv_text_vectorization/StringSplit/strided_slice_1/stackĄ
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_1Ą
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2Const*
_output_shapes
:*
dtype0*
valueB:2;
9tv_text_vectorization/StringSplit/strided_slice_1/stack_2æ
1tv_text_vectorization/StringSplit/strided_slice_1StridedSlice7tv_text_vectorization/StringSplit/StringSplitV2:shape:0@tv_text_vectorization/StringSplit/strided_slice_1/stack:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_1:output:0Btv_text_vectorization/StringSplit/strided_slice_1/stack_2:output:0*
Index0*
T0	*
_output_shapes
: *
shrink_axis_mask23
1tv_text_vectorization/StringSplit/strided_slice_1³
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CastCast8tv_text_vectorization/StringSplit/strided_slice:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2Z
Xtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast¬
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Cast:tv_text_vectorization/StringSplit/strided_slice_1:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1Ō
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ShapeShape\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0*
T0*
_output_shapes
:2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstConst*
_output_shapes
:*
dtype0*
valueB: 2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ConstÉ
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/ProdProdktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Shape:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const:output:0*
T0*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yConst*
_output_shapes
: *
dtype0*
value	B : 2h
ftv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/yÕ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/GreaterGreaterjtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Prod:output:0otv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater/y:output:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greaterč
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/CastCasthtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Greater:z:0*

DstT0*

SrcT0
*
_output_shapes
: 2c
atv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1Const*
_output_shapes
:*
dtype0*
valueB: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaxMax\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_1:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yConst*
_output_shapes
: *
dtype0*
value	B :2d
btv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/yĘ
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/addAddV2itv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Max:output:0ktv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add/y:output:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add¹
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mulMuletv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Cast:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/add:z:0*
T0*
_output_shapes
: 2b
`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul¾
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumMaximum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/mul:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MaximumĀ
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/MinimumMinimum^tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast_1:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Maximum:z:0*
T0*
_output_shapes
: 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2Const*
_output_shapes
: *
dtype0	*
valueB	 2f
dtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2æ
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/BincountBincount\tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cast:y:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Minimum:z:0mtv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Const_2:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2g
etv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axisČ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/CumsumCumsumltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/bincount/Bincount:bins:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum/axis:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0Const*
_output_shapes
:*
dtype0	*
valueB	R 2e
ctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisConst*
_output_shapes
: *
dtype0*
value	B : 2a
_tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axisµ
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concatConcatV2ltv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/values_0:output:0`tv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/Cumsum:out:0htv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat/axis:output:0*
N*
T0	*#
_output_shapes
:’’’’’’’’’2\
Ztv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Ytv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handle8tv_text_vectorization/StringSplit/StringSplitV2:values:0Ztv_text_vectorization_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*#
_output_shapes
:’’’’’’’’’2N
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2
5tv_text_vectorization/string_lookup/assert_equal/NoOpNoOp*
_output_shapes
 27
5tv_text_vectorization/string_lookup/assert_equal/NoOpķ
,tv_text_vectorization/string_lookup/IdentityIdentityUtv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
T0	*#
_output_shapes
:’’’’’’’’’2.
,tv_text_vectorization/string_lookup/Identity’
.tv_text_vectorization/string_lookup/Identity_1Identityctv_text_vectorization/StringSplit/RaggedFromValueRowIds/RowPartitionFromValueRowIds/concat:output:0*
T0	*#
_output_shapes
:’’’’’’’’’20
.tv_text_vectorization/string_lookup/Identity_1Ŗ
2tv_text_vectorization/RaggedToTensor/default_valueConst*
_output_shapes
: *
dtype0	*
value	B	 R 24
2tv_text_vectorization/RaggedToTensor/default_value£
*tv_text_vectorization/RaggedToTensor/ConstConst*
_output_shapes
: *
dtype0	*
valueB	 R
’’’’’’’’’2,
*tv_text_vectorization/RaggedToTensor/Const
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensorRaggedTensorToTensor3tv_text_vectorization/RaggedToTensor/Const:output:05tv_text_vectorization/string_lookup/Identity:output:0;tv_text_vectorization/RaggedToTensor/default_value:output:07tv_text_vectorization/string_lookup/Identity_1:output:0*
T0	*
Tindex0	*
Tshape0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’*
num_row_partition_tensors*%
row_partition_types

ROW_SPLITS2;
9tv_text_vectorization/RaggedToTensor/RaggedTensorToTensor
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2LookupTableFindV2Fsl_string_lookup_none_lookup_table_find_lookuptablefindv2_table_handleslGsl_string_lookup_none_lookup_table_find_lookuptablefindv2_default_value",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*'
_output_shapes
:’’’’’’’’’2;
9sl_string_lookup/None_lookup_table_find/LookupTableFindV2ä
concatenate/PartitionedCallPartitionedCallBtv_text_vectorization/RaggedToTensor/RaggedTensorToTensor:result:0Bsl_string_lookup/None_lookup_table_find/LookupTableFindV2:values:0*
Tin
2		*
Tout
2	*
_collective_manager_ids
 *0
_output_shapes
:’’’’’’’’’’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *M
fHRF
D__inference_concatenate_layer_call_and_return_conditional_losses_4482
concatenate/PartitionedCallh
tf.math.add/Add/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.add/Add/y¦
tf.math.add/AddAdd$concatenate/PartitionedCall:output:0tf.math.add/Add/y:output:0*
T0	*0
_output_shapes
:’’’’’’’’’’’’’’’’’’2
tf.math.add/Add
*tf.math.reduce_prod/Prod/reduction_indicesConst*
_output_shapes
: *
dtype0*
value	B :2,
*tf.math.reduce_prod/Prod/reduction_indices“
tf.math.reduce_prod/ProdProdtf.math.add/Add:z:03tf.math.reduce_prod/Prod/reduction_indices:output:0*
T0	*#
_output_shapes
:’’’’’’’’’2
tf.math.reduce_prod/Prodv
tf.math.divide/truediv/yConst*
_output_shapes
: *
dtype0	*
value	B	 R2
tf.math.divide/truediv/y¢
tf.math.divide/truediv/CastCast!tf.math.reduce_prod/Prod:output:0*

DstT0*

SrcT0	*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv/Cast
tf.math.divide/truediv/Cast_1Cast!tf.math.divide/truediv/y:output:0*

DstT0*

SrcT0	*
_output_shapes
: 2
tf.math.divide/truediv/Cast_1­
tf.math.divide/truedivRealDivtf.math.divide/truediv/Cast:y:0!tf.math.divide/truediv/Cast_1:y:0*
T0*#
_output_shapes
:’’’’’’’’’2
tf.math.divide/truediv}
tf.cast/CastCasttf.math.divide/truediv:z:0*

DstT0*

SrcT0*#
_output_shapes
:’’’’’’’’’2
tf.cast/CastÕ
expand/PartitionedCallPartitionedCalltf.cast/Cast:y:0*
Tin
2*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *H
fCRA
?__inference_expand_layer_call_and_return_conditional_losses_4782
expand/PartitionedCallž
IdentityIdentityexpand/PartitionedCall:output:0:^sl_string_lookup/None_lookup_table_find/LookupTableFindV2M^tv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 2v
9sl_string_lookup/None_lookup_table_find/LookupTableFindV29sl_string_lookup/None_lookup_table_find/LookupTableFindV22
Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2Ltv_text_vectorization/string_lookup/None_lookup_table_find/LookupTableFindV2:K G
'
_output_shapes
:’’’’’’’’’

_user_specified_nametv:KG
'
_output_shapes
:’’’’’’’’’

_user_specified_namesl:

_output_shapes
: :

_output_shapes
: 

@
$__inference_expand_layer_call_fn_973

inputs
identity½
PartitionedCallPartitionedCallinputs*
Tin
2*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *H
fCRA
?__inference_expand_layer_call_and_return_conditional_losses_4722
PartitionedCalll
IdentityIdentityPartitionedCall:output:0*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*"
_input_shapes
:’’’’’’’’’:K G
#
_output_shapes
:’’’’’’’’’
 
_user_specified_nameinputs
	
ź
__inference_restore_fn_1062
restored_tensors_0
restored_tensors_1	L
Hstring_lookup_index_table_table_restore_lookuptableimportv2_table_handle
identity¢;string_lookup_index_table_table_restore/LookupTableImportV2Ž
;string_lookup_index_table_table_restore/LookupTableImportV2LookupTableImportV2Hstring_lookup_index_table_table_restore_lookuptableimportv2_table_handlerestored_tensors_0restored_tensors_1",/job:localhost/replica:0/task:0/device:CPU:0*	
Tin0*

Tout0	*
_output_shapes
 2=
;string_lookup_index_table_table_restore/LookupTableImportV2P
ConstConst*
_output_shapes
: *
dtype0*
value	B :2
Const
IdentityIdentityConst:output:0<^string_lookup_index_table_table_restore/LookupTableImportV2*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes
:::2z
;string_lookup_index_table_table_restore/LookupTableImportV2;string_lookup_index_table_table_restore/LookupTableImportV2:L H

_output_shapes
:
,
_user_specified_namerestored_tensors_0:LH

_output_shapes
:
,
_user_specified_namerestored_tensors_1
ā
G
__inference__creator_998
identity¢string_lookup_index_table¢
string_lookup_index_tableMutableHashTableV2*
_output_shapes
: *
	key_dtype0*
shared_name	table_6*
value_dtype0	2
string_lookup_index_table
IdentityIdentity(string_lookup_index_table:table_handle:0^string_lookup_index_table*
T0*
_output_shapes
: 2

Identity"
identityIdentity:output:0*
_input_shapes 26
string_lookup_index_tablestring_lookup_index_table


!__inference_signature_wrapper_729
sl
tv
unknown
	unknown_0	
	unknown_1
	unknown_2	
identity¢StatefulPartitionedCallć
StatefulPartitionedCallStatefulPartitionedCalltvslunknown	unknown_0	unknown_1	unknown_2*
Tin

2		*
Tout
2*
_collective_manager_ids
 *'
_output_shapes
:’’’’’’’’’* 
_read_only_resource_inputs
 *-
config_proto

CPU

GPU 2J 8 *'
f"R 
__inference__wrapped_model_3882
StatefulPartitionedCall
IdentityIdentity StatefulPartitionedCall:output:0^StatefulPartitionedCall*
T0*'
_output_shapes
:’’’’’’’’’2

Identity"
identityIdentity:output:0*E
_input_shapes4
2:’’’’’’’’’:’’’’’’’’’:: :: 22
StatefulPartitionedCallStatefulPartitionedCall:K G
'
_output_shapes
:’’’’’’’’’

_user_specified_namesl:KG
'
_output_shapes
:’’’’’’’’’

_user_specified_nametv:

_output_shapes
: :

_output_shapes
: "±L
saver_filename:0StatefulPartitionedCall_1:0StatefulPartitionedCall_28"
saved_model_main_op

NoOp*>
__saved_model_init_op%#
__saved_model_init_op

NoOp*Ņ
serving_default¾
1
sl+
serving_default_sl:0’’’’’’’’’
1
tv+
serving_default_tv:0’’’’’’’’’:
expand0
StatefulPartitionedCall:0’’’’’’’’’tensorflow/serving/predict:ø
Š8
layer-0
layer-1
layer_with_weights-0
layer-2
layer_with_weights-1
layer-3
layer-4
layer-5
layer-6
layer-7
	layer-8

layer-9
regularization_losses
trainable_variables
	variables
	keras_api

signatures
*4&call_and_return_all_conditional_losses
5__call__
6_default_save_signature"Ž5
_tf_keras_networkĀ5{"class_name": "Functional", "name": "vectorization_float", "trainable": true, "expects_training_arg": true, "dtype": "float32", "batch_input_shape": null, "must_restore_from_config": false, "config": {"name": "vectorization_float", "layers": [{"class_name": "InputLayer", "config": {"batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "dtype": "string", "sparse": false, "ragged": false, "name": "tv"}, "name": "tv", "inbound_nodes": []}, {"class_name": "InputLayer", "config": {"batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "dtype": "string", "sparse": false, "ragged": false, "name": "sl"}, "name": "sl", "inbound_nodes": []}, {"class_name": "TextVectorization", "config": {"name": "tv_text_vectorization", "trainable": true, "dtype": "string", "max_tokens": null, "standardize": "lower_and_strip_punctuation", "split": "whitespace", "ngrams": null, "output_mode": "int", "output_sequence_length": null, "pad_to_max_tokens": true}, "name": "tv_text_vectorization", "inbound_nodes": [[["tv", 0, 0, {}]]]}, {"class_name": "StringLookup", "config": {"name": "sl_string_lookup", "trainable": true, "batch_input_shape": {"class_name": "__tuple__", "items": [null]}, "dtype": "string", "invert": false, "max_tokens": null, "num_oov_indices": 1, "oov_token": "[UNK]", "mask_token": "", "encoding": "utf-8"}, "name": "sl_string_lookup", "inbound_nodes": [[["sl", 0, 0, {}]]]}, {"class_name": "Concatenate", "config": {"name": "concatenate", "trainable": true, "dtype": "float32", "axis": -1}, "name": "concatenate", "inbound_nodes": [[["tv_text_vectorization", 0, 0, {}], ["sl_string_lookup", 0, 0, {}]]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.math.add", "trainable": true, "dtype": "float32", "function": "math.add"}, "name": "tf.math.add", "inbound_nodes": [["concatenate", 0, 0, {"y": 1, "name": "add_one"}]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.math.reduce_prod", "trainable": true, "dtype": "float32", "function": "math.reduce_prod"}, "name": "tf.math.reduce_prod", "inbound_nodes": [["tf.math.add", 0, 0, {"axis": 1, "name": "reduce_prod"}]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.math.divide", "trainable": true, "dtype": "float32", "function": "math.divide"}, "name": "tf.math.divide", "inbound_nodes": [["tf.math.reduce_prod", 0, 0, {"y": 7, "name": "divide"}]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.cast", "trainable": true, "dtype": "float32", "function": "cast"}, "name": "tf.cast", "inbound_nodes": [["tf.math.divide", 0, 0, {"dtype": "float32", "name": "cast_f32"}]]}, {"class_name": "Lambda", "config": {"name": "expand", "trainable": true, "dtype": "float32", "function": {"class_name": "__tuple__", "items": ["4wEAAAAAAAAAAQAAAAQAAABDAAAAcw4AAAB0AGoBfABkAWQCjQJTACkDTukBAAAAKQHaBGF4aXMp\nAtoCdGbaC2V4cGFuZF9kaW1zKQHaAW+pAHIGAAAA+hNtYWtlX21vY2tfbW9kZWxzLnB52gg8bGFt\nYmRhProAAABzAAAAAA==\n", null, null]}, "function_type": "lambda", "module": "__main__", "output_shape": null, "output_shape_type": "raw", "output_shape_module": null, "arguments": {}}, "name": "expand", "inbound_nodes": [[["tf.cast", 0, 0, {}]]]}], "input_layers": [["tv", 0, 0], ["sl", 0, 0]], "output_layers": [["expand", 0, 0]]}, "input_spec": [{"class_name": "InputSpec", "config": {"dtype": null, "shape": {"class_name": "__tuple__", "items": [null, 1]}, "ndim": 2, "max_ndim": null, "min_ndim": null, "axes": {}}}, {"class_name": "InputSpec", "config": {"dtype": null, "shape": {"class_name": "__tuple__", "items": [null, 1]}, "ndim": 2, "max_ndim": null, "min_ndim": null, "axes": {}}}], "build_input_shape": [{"class_name": "TensorShape", "items": [null, 1]}, {"class_name": "TensorShape", "items": [null, 1]}], "is_graph_network": true, "keras_version": "2.4.0", "backend": "tensorflow", "model_config": {"class_name": "Functional", "config": {"name": "vectorization_float", "layers": [{"class_name": "InputLayer", "config": {"batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "dtype": "string", "sparse": false, "ragged": false, "name": "tv"}, "name": "tv", "inbound_nodes": []}, {"class_name": "InputLayer", "config": {"batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "dtype": "string", "sparse": false, "ragged": false, "name": "sl"}, "name": "sl", "inbound_nodes": []}, {"class_name": "TextVectorization", "config": {"name": "tv_text_vectorization", "trainable": true, "dtype": "string", "max_tokens": null, "standardize": "lower_and_strip_punctuation", "split": "whitespace", "ngrams": null, "output_mode": "int", "output_sequence_length": null, "pad_to_max_tokens": true}, "name": "tv_text_vectorization", "inbound_nodes": [[["tv", 0, 0, {}]]]}, {"class_name": "StringLookup", "config": {"name": "sl_string_lookup", "trainable": true, "batch_input_shape": {"class_name": "__tuple__", "items": [null]}, "dtype": "string", "invert": false, "max_tokens": null, "num_oov_indices": 1, "oov_token": "[UNK]", "mask_token": "", "encoding": "utf-8"}, "name": "sl_string_lookup", "inbound_nodes": [[["sl", 0, 0, {}]]]}, {"class_name": "Concatenate", "config": {"name": "concatenate", "trainable": true, "dtype": "float32", "axis": -1}, "name": "concatenate", "inbound_nodes": [[["tv_text_vectorization", 0, 0, {}], ["sl_string_lookup", 0, 0, {}]]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.math.add", "trainable": true, "dtype": "float32", "function": "math.add"}, "name": "tf.math.add", "inbound_nodes": [["concatenate", 0, 0, {"y": 1, "name": "add_one"}]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.math.reduce_prod", "trainable": true, "dtype": "float32", "function": "math.reduce_prod"}, "name": "tf.math.reduce_prod", "inbound_nodes": [["tf.math.add", 0, 0, {"axis": 1, "name": "reduce_prod"}]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.math.divide", "trainable": true, "dtype": "float32", "function": "math.divide"}, "name": "tf.math.divide", "inbound_nodes": [["tf.math.reduce_prod", 0, 0, {"y": 7, "name": "divide"}]]}, {"class_name": "TFOpLambda", "config": {"name": "tf.cast", "trainable": true, "dtype": "float32", "function": "cast"}, "name": "tf.cast", "inbound_nodes": [["tf.math.divide", 0, 0, {"dtype": "float32", "name": "cast_f32"}]]}, {"class_name": "Lambda", "config": {"name": "expand", "trainable": true, "dtype": "float32", "function": {"class_name": "__tuple__", "items": ["4wEAAAAAAAAAAQAAAAQAAABDAAAAcw4AAAB0AGoBfABkAWQCjQJTACkDTukBAAAAKQHaBGF4aXMp\nAtoCdGbaC2V4cGFuZF9kaW1zKQHaAW+pAHIGAAAA+hNtYWtlX21vY2tfbW9kZWxzLnB52gg8bGFt\nYmRhProAAABzAAAAAA==\n", null, null]}, "function_type": "lambda", "module": "__main__", "output_shape": null, "output_shape_type": "raw", "output_shape_module": null, "arguments": {}}, "name": "expand", "inbound_nodes": [[["tf.cast", 0, 0, {}]]]}], "input_layers": [["tv", 0, 0], ["sl", 0, 0]], "output_layers": [["expand", 0, 0]]}}}
Ż"Ś
_tf_keras_input_layerŗ{"class_name": "InputLayer", "name": "tv", "dtype": "string", "sparse": false, "ragged": false, "batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "config": {"batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "dtype": "string", "sparse": false, "ragged": false, "name": "tv"}}
Ż"Ś
_tf_keras_input_layerŗ{"class_name": "InputLayer", "name": "sl", "dtype": "string", "sparse": false, "ragged": false, "batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "config": {"batch_input_shape": {"class_name": "__tuple__", "items": [null, 1]}, "dtype": "string", "sparse": false, "ragged": false, "name": "sl"}}

state_variables
_index_lookup_layer
	keras_api"Å
_tf_keras_layer«{"class_name": "TextVectorization", "name": "tv_text_vectorization", "trainable": true, "expects_training_arg": false, "dtype": "string", "batch_input_shape": null, "stateful": false, "must_restore_from_config": true, "config": {"name": "tv_text_vectorization", "trainable": true, "dtype": "string", "max_tokens": null, "standardize": "lower_and_strip_punctuation", "split": "whitespace", "ngrams": null, "output_mode": "int", "output_sequence_length": null, "pad_to_max_tokens": true}, "build_input_shape": {"class_name": "TensorShape", "items": [4, 1]}}
Ķ
state_variables

_table
	keras_api"
_tf_keras_layer{"class_name": "StringLookup", "name": "sl_string_lookup", "trainable": true, "expects_training_arg": false, "dtype": "string", "batch_input_shape": {"class_name": "__tuple__", "items": [null]}, "stateful": false, "must_restore_from_config": true, "config": {"name": "sl_string_lookup", "trainable": true, "batch_input_shape": {"class_name": "__tuple__", "items": [null]}, "dtype": "string", "invert": false, "max_tokens": null, "num_oov_indices": 1, "oov_token": "[UNK]", "mask_token": "", "encoding": "utf-8"}}
Ź
regularization_losses
trainable_variables
	variables
	keras_api
*;&call_and_return_all_conditional_losses
<__call__"»
_tf_keras_layer”{"class_name": "Concatenate", "name": "concatenate", "trainable": true, "expects_training_arg": false, "dtype": "float32", "batch_input_shape": null, "stateful": false, "must_restore_from_config": false, "config": {"name": "concatenate", "trainable": true, "dtype": "float32", "axis": -1}, "build_input_shape": [{"class_name": "TensorShape", "items": [null, null]}, {"class_name": "TensorShape", "items": [null, 1]}]}
×
	keras_api"Å
_tf_keras_layer«{"class_name": "TFOpLambda", "name": "tf.math.add", "trainable": true, "expects_training_arg": false, "dtype": "float32", "batch_input_shape": null, "stateful": false, "must_restore_from_config": true, "config": {"name": "tf.math.add", "trainable": true, "dtype": "float32", "function": "math.add"}}
ļ
	keras_api"Ż
_tf_keras_layerĆ{"class_name": "TFOpLambda", "name": "tf.math.reduce_prod", "trainable": true, "expects_training_arg": false, "dtype": "float32", "batch_input_shape": null, "stateful": false, "must_restore_from_config": true, "config": {"name": "tf.math.reduce_prod", "trainable": true, "dtype": "float32", "function": "math.reduce_prod"}}
ą
	keras_api"Ī
_tf_keras_layer“{"class_name": "TFOpLambda", "name": "tf.math.divide", "trainable": true, "expects_training_arg": false, "dtype": "float32", "batch_input_shape": null, "stateful": false, "must_restore_from_config": true, "config": {"name": "tf.math.divide", "trainable": true, "dtype": "float32", "function": "math.divide"}}
Ė
	keras_api"¹
_tf_keras_layer{"class_name": "TFOpLambda", "name": "tf.cast", "trainable": true, "expects_training_arg": false, "dtype": "float32", "batch_input_shape": null, "stateful": false, "must_restore_from_config": true, "config": {"name": "tf.cast", "trainable": true, "dtype": "float32", "function": "cast"}}
µ
regularization_losses
trainable_variables
 	variables
!	keras_api
*=&call_and_return_all_conditional_losses
>__call__"¦
_tf_keras_layer{"class_name": "Lambda", "name": "expand", "trainable": true, "expects_training_arg": true, "dtype": "float32", "batch_input_shape": null, "stateful": false, "must_restore_from_config": false, "config": {"name": "expand", "trainable": true, "dtype": "float32", "function": {"class_name": "__tuple__", "items": ["4wEAAAAAAAAAAQAAAAQAAABDAAAAcw4AAAB0AGoBfABkAWQCjQJTACkDTukBAAAAKQHaBGF4aXMp\nAtoCdGbaC2V4cGFuZF9kaW1zKQHaAW+pAHIGAAAA+hNtYWtlX21vY2tfbW9kZWxzLnB52gg8bGFt\nYmRhProAAABzAAAAAA==\n", null, null]}, "function_type": "lambda", "module": "__main__", "output_shape": null, "output_shape_type": "raw", "output_shape_module": null, "arguments": {}}}
 "
trackable_list_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
Ź
"metrics
#layer_metrics
$non_trainable_variables
%layer_regularization_losses

&layers
regularization_losses
trainable_variables
	variables
5__call__
6_default_save_signature
*4&call_and_return_all_conditional_losses
&4"call_and_return_conditional_losses"
_generic_user_object
,
?serving_default"
signature_map
 "
trackable_dict_wrapper
Ó
'state_variables

(_table
)	keras_api" 
_tf_keras_layer{"class_name": "StringLookup", "name": "string_lookup", "trainable": true, "expects_training_arg": false, "dtype": "string", "batch_input_shape": {"class_name": "__tuple__", "items": [null, null]}, "stateful": false, "must_restore_from_config": true, "config": {"name": "string_lookup", "trainable": true, "batch_input_shape": {"class_name": "__tuple__", "items": [null, null]}, "dtype": "string", "invert": false, "max_tokens": null, "num_oov_indices": 1, "oov_token": "[UNK]", "mask_token": "", "encoding": "utf-8"}}
"
_generic_user_object
 "
trackable_dict_wrapper
O
@_create_resource
A_initialize
B_destroy_resourceR Z
table78
"
_generic_user_object
 "
trackable_list_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
­
*metrics
+layer_metrics
,non_trainable_variables
-layer_regularization_losses

.layers
regularization_losses
trainable_variables
	variables
<__call__
*;&call_and_return_all_conditional_losses
&;"call_and_return_conditional_losses"
_generic_user_object
"
_generic_user_object
"
_generic_user_object
"
_generic_user_object
"
_generic_user_object
 "
trackable_list_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
­
/metrics
0layer_metrics
1non_trainable_variables
2layer_regularization_losses

3layers
regularization_losses
trainable_variables
 	variables
>__call__
*=&call_and_return_all_conditional_losses
&="call_and_return_conditional_losses"
_generic_user_object
 "
trackable_list_wrapper
 "
trackable_dict_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
f
0
1
2
3
4
5
6
7
	8

9"
trackable_list_wrapper
 "
trackable_dict_wrapper
O
C_create_resource
D_initialize
E_destroy_resourceR Z
table9:
"
_generic_user_object
 "
trackable_list_wrapper
 "
trackable_dict_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
 "
trackable_dict_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
 "
trackable_list_wrapper
ž2ū
L__inference_vectorization_float_layer_call_and_return_conditional_losses_492
L__inference_vectorization_float_layer_call_and_return_conditional_losses_915
L__inference_vectorization_float_layer_call_and_return_conditional_losses_849
L__inference_vectorization_float_layer_call_and_return_conditional_losses_556Ą
·²³
FullArgSpec1
args)&
jself
jinputs

jtraining
jmask
varargs
 
varkw
 
defaults
p 

 

kwonlyargs 
kwonlydefaultsŖ 
annotationsŖ *
 
2
1__inference_vectorization_float_layer_call_fn_713
1__inference_vectorization_float_layer_call_fn_635
1__inference_vectorization_float_layer_call_fn_943
1__inference_vectorization_float_layer_call_fn_929Ą
·²³
FullArgSpec1
args)&
jself
jinputs

jtraining
jmask
varargs
 
varkw
 
defaults
p 

 

kwonlyargs 
kwonlydefaultsŖ 
annotationsŖ *
 
ś2÷
__inference__wrapped_model_388Ō
²
FullArgSpec
args 
varargsjargs
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *D¢A
?<

tv’’’’’’’’’

sl’’’’’’’’’
ÜBŁ
__inference_save_fn_1027checkpoint_key"Ŗ
²
FullArgSpec
args
jcheckpoint_key
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢	
 
B’
__inference_restore_fn_1035restored_tensors_0restored_tensors_1"µ
²
FullArgSpec
args 
varargsjrestored_tensors
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢
	
		
ÜBŁ
__inference_save_fn_1054checkpoint_key"Ŗ
²
FullArgSpec
args
jcheckpoint_key
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢	
 
B’
__inference_restore_fn_1062restored_tensors_0restored_tensors_1"µ
²
FullArgSpec
args 
varargsjrestored_tensors
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢
	
		
ī2ė
D__inference_concatenate_layer_call_and_return_conditional_losses_950¢
²
FullArgSpec
args
jself
jinputs
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *
 
Ó2Š
)__inference_concatenate_layer_call_fn_956¢
²
FullArgSpec
args
jself
jinputs
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *
 
Č2Å
?__inference_expand_layer_call_and_return_conditional_losses_968
?__inference_expand_layer_call_and_return_conditional_losses_962Ą
·²³
FullArgSpec1
args)&
jself
jinputs
jmask

jtraining
varargs
 
varkw
 
defaults

 
p 

kwonlyargs 
kwonlydefaultsŖ 
annotationsŖ *
 
2
$__inference_expand_layer_call_fn_973
$__inference_expand_layer_call_fn_978Ą
·²³
FullArgSpec1
args)&
jself
jinputs
jmask

jtraining
varargs
 
varkw
 
defaults

 
p 

kwonlyargs 
kwonlydefaultsŖ 
annotationsŖ *
 
ÅBĀ
!__inference_signature_wrapper_729sltv"
²
FullArgSpec
args 
varargs
 
varkwjkwargs
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *
 
Æ2¬
__inference__creator_983
²
FullArgSpec
args 
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢ 
³2°
__inference__initializer_988
²
FullArgSpec
args 
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢ 
±2®
__inference__destroyer_993
²
FullArgSpec
args 
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢ 
Æ2¬
__inference__creator_998
²
FullArgSpec
args 
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢ 
“2±
__inference__initializer_1003
²
FullArgSpec
args 
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢ 
²2Æ
__inference__destroyer_1008
²
FullArgSpec
args 
varargs
 
varkw
 
defaults
 

kwonlyargs 
kwonlydefaults
 
annotationsŖ *¢ 
	J
Const
J	
Const_14
__inference__creator_983¢

¢ 
Ŗ " 4
__inference__creator_998¢

¢ 
Ŗ " 7
__inference__destroyer_1008¢

¢ 
Ŗ " 6
__inference__destroyer_993¢

¢ 
Ŗ " 9
__inference__initializer_1003¢

¢ 
Ŗ " 8
__inference__initializer_988¢

¢ 
Ŗ " Ŗ
__inference__wrapped_model_388(FGN¢K
D¢A
?<

tv’’’’’’’’’

sl’’’’’’’’’
Ŗ "/Ŗ,
*
expand 
expand’’’’’’’’’Ž
D__inference_concatenate_layer_call_and_return_conditional_losses_950c¢`
Y¢V
TQ
+(
inputs/0’’’’’’’’’’’’’’’’’’	
"
inputs/1’’’’’’’’’	
Ŗ ".¢+
$!
0’’’’’’’’’’’’’’’’’’	
 ¶
)__inference_concatenate_layer_call_fn_956c¢`
Y¢V
TQ
+(
inputs/0’’’’’’’’’’’’’’’’’’	
"
inputs/1’’’’’’’’’	
Ŗ "!’’’’’’’’’’’’’’’’’’	
?__inference_expand_layer_call_and_return_conditional_losses_962\3¢0
)¢&

inputs’’’’’’’’’

 
p
Ŗ "%¢"

0’’’’’’’’’
 
?__inference_expand_layer_call_and_return_conditional_losses_968\3¢0
)¢&

inputs’’’’’’’’’

 
p 
Ŗ "%¢"

0’’’’’’’’’
 w
$__inference_expand_layer_call_fn_973O3¢0
)¢&

inputs’’’’’’’’’

 
p
Ŗ "’’’’’’’’’w
$__inference_expand_layer_call_fn_978O3¢0
)¢&

inputs’’’’’’’’’

 
p 
Ŗ "’’’’’’’’’x
__inference_restore_fn_1035YK¢H
A¢>

restored_tensors_0

restored_tensors_1	
Ŗ " x
__inference_restore_fn_1062Y(K¢H
A¢>

restored_tensors_0

restored_tensors_1	
Ŗ " 
__inference_save_fn_1027ö&¢#
¢

checkpoint_key 
Ŗ "ČÄ
`Ŗ]

name
0/name 
#

slice_spec
0/slice_spec 

tensor
0/tensor
`Ŗ]

name
1/name 
#

slice_spec
1/slice_spec 

tensor
1/tensor	
__inference_save_fn_1054ö(&¢#
¢

checkpoint_key 
Ŗ "ČÄ
`Ŗ]

name
0/name 
#

slice_spec
0/slice_spec 

tensor
0/tensor
`Ŗ]

name
1/name 
#

slice_spec
1/slice_spec 

tensor
1/tensor	“
!__inference_signature_wrapper_729(FGU¢R
¢ 
KŖH
"
sl
sl’’’’’’’’’
"
tv
tv’’’’’’’’’"/Ŗ,
*
expand 
expand’’’’’’’’’Ö
L__inference_vectorization_float_layer_call_and_return_conditional_losses_492(FGV¢S
L¢I
?<

tv’’’’’’’’’

sl’’’’’’’’’
p

 
Ŗ "%¢"

0’’’’’’’’’
 Ö
L__inference_vectorization_float_layer_call_and_return_conditional_losses_556(FGV¢S
L¢I
?<

tv’’’’’’’’’

sl’’’’’’’’’
p 

 
Ŗ "%¢"

0’’’’’’’’’
 ā
L__inference_vectorization_float_layer_call_and_return_conditional_losses_849(FGb¢_
X¢U
KH
"
inputs/0’’’’’’’’’
"
inputs/1’’’’’’’’’
p

 
Ŗ "%¢"

0’’’’’’’’’
 ā
L__inference_vectorization_float_layer_call_and_return_conditional_losses_915(FGb¢_
X¢U
KH
"
inputs/0’’’’’’’’’
"
inputs/1’’’’’’’’’
p 

 
Ŗ "%¢"

0’’’’’’’’’
 ­
1__inference_vectorization_float_layer_call_fn_635x(FGV¢S
L¢I
?<

tv’’’’’’’’’

sl’’’’’’’’’
p

 
Ŗ "’’’’’’’’’­
1__inference_vectorization_float_layer_call_fn_713x(FGV¢S
L¢I
?<

tv’’’’’’’’’

sl’’’’’’’’’
p 

 
Ŗ "’’’’’’’’’ŗ
1__inference_vectorization_float_layer_call_fn_929(FGb¢_
X¢U
KH
"
inputs/0’’’’’’’’’
"
inputs/1’’’’’’’’’
p

 
Ŗ "’’’’’’’’’ŗ
1__inference_vectorization_float_layer_call_fn_943(FGb¢_
X¢U
KH
"
inputs/0’’’’’’’’’
"
inputs/1’’’’’’’’’
p 

 
Ŗ "’’’’’’’’’