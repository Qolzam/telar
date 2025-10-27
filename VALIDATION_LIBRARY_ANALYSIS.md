# Validation Library Analysis: Yup vs Zod

## Current Situation

### In Telar New Arch (`apps/web`)

**BOTH libraries are installed**:
- `yup: ^1.7.1` - Used in Login and Signup forms
- `zod: ^4.1.11` - Used in all Settings/Account forms

**Current Usage**:
1. **Yup** (with Formik):
   - `LoginForm.tsx` - Login validation
   - `SignupForm.tsx` - Registration validation
   - Uses `formik` + `yup` integration

2. **Zod** (with React Hook Form):
   - `ChangePasswordForm.tsx` - Password change
   - `SocialLinksForm.tsx` - Social media links
   - `AccountGeneralForm.tsx` - User profile
   - Uses `react-hook-form` + `zodResolver`

### In Minimal JavaScript v6.0.1 Reference

**Uses ONLY Zod**:
- `zod: ^3.23.8` (older version)
- All forms use `react-hook-form` + `zod`
- No Yup in the project

---

## Detailed Comparison

### Yup
**Pros**:
- Older, battle-tested library
- Good Formik integration
- Simple, chainable API
- Good for quick forms

**Cons**:
- No TypeScript inference (requires manual types)
- Larger bundle size (~49 KB minified)
- Performance slower on large forms
- Less type-safe
- Maintenance declining

**Current Usage**: 2 forms (Login, Signup)

### Zod
**Pros**:
- **Full TypeScript support** with automatic type inference
- Better performance
- Smaller bundle size (~12 KB minified)
- More modern API design
- Actively maintained
- Can be used for API validation, forms, and more
- Strong TypeScript integration

**Cons**:
- Newer library (but very stable)
- Learning curve if coming from Yup

**Current Usage**: 3 forms (Password, Social, General)

---

## Why This Is Problematic

### 1. **Consistency Issue**
- Developers need to know TWO different APIs
- Different validation syntax across codebase
- Harder to maintain and onboard new developers

### 2. **Bundle Size**
- Installing both libraries adds unnecessary weight
- Yup: ~49 KB, Zod: ~12 KB
- **Total waste**: ~49 KB (since Zod replaces Yup)

### 3. **Form Library Mismatch**
- Yup used with Formik
- Zod used with React Hook Form
- Inconsistent form management patterns

### 4. **Type Safety**
- Yup forms require manual TypeScript types
- Zod automatically infers types from schema
- Mixed approach causes type inconsistencies

### 5. **Maintenance Burden**
- Two libraries to update
- Two different error handling approaches
- Two different testing strategies

---

## Recommended Solution

### **Migrate everything to Zod + React Hook Form**

**Why?**
1. **Reference implementation uses Zod exclusively**
2. **Better TypeScript support** (automatic type inference)
3. **Smaller bundle size** (37 KB savings)
4. **More performant** (especially on large forms)
5. **Modern, actively maintained** library
6. **Single API to learn**

### Migration Plan

**Phase 1: Convert LoginForm**
```typescript
// From: Yup + Formik
const LoginSchema = Yup.object().shape({
  email: Yup.string().email().required(),
  password: Yup.string().required(),
});

// To: Zod + React Hook Form
const loginSchema = z.object({
  email: z.string().email('Invalid email').min(1, 'Required'),
  password: z.string().min(1, 'Required'),
});

type LoginFormData = z.infer<typeof loginSchema>;
```

**Phase 2: Convert SignupForm**
```typescript
// From: Yup + Formik
const RegisterSchema = Yup.object().shape({
  firstName: Yup.string().min(2).max(50).required(),
  // ...
});

// To: Zod + React Hook Form
const signupSchema = z.object({
  firstName: z.string().min(2).max(50),
  // ...
});
```

**Phase 3: Remove Yup**
```bash
pnpm remove yup @types/yup
```

---

## Action Items

### Immediate Actions (High Priority)

1. **Convert LoginForm.tsx to Zod + React Hook Form**
   - Replace Yup schema with Zod
   - Replace Formik with React Hook Form
   - Update form handling logic

2. **Convert SignupForm.tsx to Zod + React Hook Form**
   - Replace Yup schema with Zod
   - Replace Formik with React Hook Form
   - Update validation messages to use translations

3. **Remove Yup dependencies**
   - `pnpm remove yup`
   - `pnpm remove @types/yup` (if installed)
   - Remove all Yup imports

### Benefits After Migration

✅ Single validation library (Zod)  
✅ Consistent form management (React Hook Form)  
✅ Better TypeScript support (automatic inference)  
✅ 37 KB bundle size reduction  
✅ Better performance  
✅ Easier maintenance  
✅ Matches reference implementation  

### Potential Breaking Changes

⚠️ **Formik-specific features** need to be replaced:
- Form state management (React Hook Form handles this)
- Field arrays (React Hook Form has `useFieldArray`)
- Form-level validation timing (React Hook Form has `mode` options)

---

## Code Comparison

### Before (Yup + Formik)
```typescript
import * as Yup from 'yup';
import { useFormik, FormikProvider } from 'formik';

const LoginSchema = Yup.object().shape({
  email: Yup.string()
    .email(t('validation:email.invalid'))
    .required(t('validation:email.required')),
  password: Yup.string()
    .required(t('validation:password.required')),
});

const formik = useFormik({
  initialValues: { email: '', password: '' },
  validationSchema: LoginSchema,
  onSubmit: async (values) => { /* ... */ }
});

// Usage
<FormikProvider value={formik}>
  <Box component={Form}>
    <TextField {...getFieldProps('email')} />
    <TextField {...getFieldProps('password')} />
  </Box>
</FormikProvider>
```

### After (Zod + React Hook Form)
```typescript
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';

const loginSchema = z.object({
  email: z.string()
    .email(t('validation:email.invalid'))
    .min(1, t('validation:email.required')),
  password: z.string()
    .min(1, t('validation:password.required')),
});

type LoginFormData = z.infer<typeof loginSchema>;

const {
  register,
  handleSubmit,
  formState: { errors }
} = useForm<LoginFormData>({
  resolver: zodResolver(loginSchema)
});

// Usage
<form onSubmit={handleSubmit(onSubmit)}>
  <TextField {...register('email')} />
  <TextField {...register('password')} />
</form>
```

---

## Recommendation Priority

**CRITICAL**: This should be done soon because:
1. Current implementation is inconsistent
2. Bundle size is unnecessarily large
3. Maintainability suffers
4. Type safety is compromised

**Effort**: Medium
- 2 forms to convert (Login, Signup)
- Estimated time: 2-3 hours
- Risk: Low (well-established migration path)

**Impact**: High
- Better developer experience
- Smaller bundle
- Consistent codebase
- Better performance

---

## Conclusion

**Current State**: Redundant use of both Yup and Zod  
**Problem**: Inconsistency, extra bundle size, maintenance burden  
**Solution**: Migrate to Zod + React Hook Form exclusively  
**Reference**: Minimal JS v6 uses Zod exclusively  
**Priority**: High (should be done in next sprint)
