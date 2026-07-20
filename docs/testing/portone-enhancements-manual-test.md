# PortOne Enhancements Manual Testing Checklist

## Prerequisites
- Server running with latest code
- Admin access  
- Browser with console access

## Test Suite 1: Debug Mode

### Test 1.1: Debug Mode Disabled
- [ ] Verify debug_enabled = false in database
- [ ] Open browser console on storefront
- [ ] Proceed through PortOne payment
- [ ] **Expected**: NO console logs

### Test 1.2: Debug Mode Enabled  
- [ ] Set debug_enabled = true in database
- [ ] Proceed through PortOne payment
- [ ] **Expected**: Console logs appear

## Test Suite 2: Currency Restrictions

### Test 2.1: Currency Match
- [ ] Set currency to KRW, PortOne supports KRW
- [ ] **Expected**: PortOne appears

### Test 2.2: Currency Mismatch
- [ ] Set currency to USD, PortOne supports only KRW
- [ ] **Expected**: PortOne hidden

## Test Suite 3: Currency Formatting  

### Test 3.1: KRW Korean Locale
- [ ] Language: ko, Currency: KRW, Amount: 300000
- [ ] **Expected**: "3,000원"

### Test 3.2: KRW English Locale
- [ ] Language: en, Currency: KRW
- [ ] **Expected**: "₩3,000"

## Test Suite 4: Korean Translation
- [ ] Language: ko
- [ ] **Expected**: PortOne shows "간편결제 서비스"

## Sign-off
- [ ] All tests passed
**Tester**: _____ **Date**: _____ **Result**: _____
