# Compatibility matrix

This document provides the information around service support for different versions of OASF schema.
It answers the question:
_what OASF versions are supported in a given OASF-SDK service_.

## Version Support

### Reading Support (Input)

Services can **read/consume** records from these OASF versions:

| OASF-SDK           | OASF v1alpha1<br>(0.7.0) | OASF v1alpha2<br>(0.8.0) | OASF v1<br>(1.0.0) |
| ------------------ | ------------------------ | ------------------------ | ------------------ |
| DecodingService    | ✅                       | ✅                       | ✅                 |
| TranslationService | ✅                       | ✅                       | ✅                 |
| ValidationService  | ✅                       | ✅                       | ✅                 |

### Writing Support (Output)

Services that can **generate/write** OASF records produce them in these versions:

| OASF-SDK           | OASF v1alpha1<br>(0.7.0) | OASF v1alpha2<br>(0.8.0) | OASF v1<br>(1.0.0) | Notes                                      |
| ------------------ | ------------------------ | ------------------------ | ------------------ | ------------------------------------------ |
| DecodingService    | N/A                      | N/A                      | N/A                | Outputs typed protobuf, not OASF records   |
| TranslationService | ❌                       | ❌                       | ✅                 | `*ToRecord` methods generate 1.0.0 only    |
| ValidationService  | N/A                      | N/A                      | N/A                | Validation only, no record generation      |
