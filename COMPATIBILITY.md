# Compatibility matrix

This document provides the information around service support for different versions of OASF schema.
It answers the question:
_what OASF versions are supported in a given OASF-SDK service_.

## Version Support

### Version Range Support

The SDK supports **version ranges** based on major.minor version numbers. This means:
- **0.7.x** versions (0.7.0, 0.7.1, etc.) → use v1alpha1 proto
- **0.8.x** versions (0.8.0, 0.8.1, etc.) → use v1alpha2 proto
- **1.x.x** versions (1.0.0, 1.0.1, 1.1.0, etc.) → use v1 proto

**Important**: The SDK only needs updates when there are proto changes, which occur with major version changes (e.g., 1.x.x → 2.x.x). Minor and patch releases within the same major version (e.g., 1.0.0 → 1.0.1 → 1.1.0) work automatically without SDK updates.

### Reading Support (Input)

Services can **read/consume** records from these OASF version ranges:

| OASF-SDK           | OASF v1alpha1<br>(0.7.x) | OASF v1alpha2<br>(0.8.x) | OASF v1<br>(1.x.x) |
| ------------------ | ------------------------ | ------------------------ | ------------------ |
| DecodingService    | ✅                       | ✅                       | ✅                 |
| TranslationService | ✅                       | ✅                       | ✅                 |
| ValidationService  | ✅                       | ✅                       | ✅                 |

### Writing Support (Output)

Services that can **generate/write** OASF records produce them in these versions:

| OASF-SDK           | OASF v1alpha1<br>(0.7.x) | OASF v1alpha2<br>(0.8.x) | OASF v1<br>(1.x.x) | Notes                                      |
| ------------------ | ------------------------ | ------------------------ | ------------------ | ------------------------------------------ |
| DecodingService    | N/A                      | N/A                      | N/A                | Outputs typed protobuf, not OASF records   |
| TranslationService | ❌                       | ❌                       | ✅                 | `*ToRecord` methods generate 1.0.0 only    |
| ValidationService  | N/A                      | N/A                      | N/A                | Validation only, no record generation      |
