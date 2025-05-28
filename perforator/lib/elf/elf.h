#pragma once

#include <util/generic/string.h>
#include <util/generic/hash.h>
#include <util/generic/vector.h>
#include <util/generic/maybe.h>
#include <util/generic/array_ref.h>

#include <llvm/Object/ObjectFile.h>
#include <llvm/Object/ELFObjectFile.h>

#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

namespace NPerforator::NELF {

struct TLocation {
    ui64 Address = 0;
    ui64 Size = 0;
};

namespace NSections {

constexpr TStringBuf kTextSectionName = ".text";
constexpr TStringBuf kDataSectionName = ".data";
constexpr TStringBuf kBssSectionName = ".bss";
constexpr TStringBuf kRoDataSectionName = ".rodata";

} // namespace NPerforator::NELF::NSections

using TSymbolMap = THashMap<TStringBuf, TLocation>;

namespace NPrivate {

template <typename Container>
TSymbolMap ParseSymbolsImpl(
    const llvm::object::ELFObjectFileBase::elf_symbol_iterator_range& symbols,
    const Container& targetSymbols
) {
    TSymbolMap result;

    for (const auto& symbol : symbols) {
        TLocation location;

        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });
        Y_LLVM_UNWRAP(address, symbol.getAddress(), { continue; });

        location.Address = address;
        location.Size = symbol.getSize();

        TStringBuf symbolName{name.data(), name.size()};
        for (const auto& targetSymbol : targetSymbols) {
            if (symbolName == targetSymbol) {
                result[symbolName] = location;
                break;
            }
        }
    }

    return result;
}

template <typename ELFT, typename Container>
TSymbolMap ParseDynsym(const llvm::object::ELFObjectFile<ELFT>& elf, const Container& symbols) {
    return ParseSymbolsImpl(elf.getDynamicSymbolIterators(), symbols);
}

template <typename ELFT, typename Container>
TSymbolMap ParseSymtab(const llvm::object::ELFObjectFile<ELFT>& elf, const Container& symbols) {
    return ParseSymbolsImpl(elf.symbols(), symbols);
}

TMaybe<TSymbolMap> RetrieveSymbolsFromDynsym(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbolsFromSymtab(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

} // namespace NPerforator::NELF::NPrivate


template <typename... Args>
TMaybe<TSymbolMap> RetrieveSymbolsFromDynsym(const llvm::object::ObjectFile& file, Args... symbols) {
    return NPerforator::NELF::NPrivate::RetrieveSymbolsFromDynsym(file, {symbols...});
}

template <typename... Args>
TMaybe<TSymbolMap> RetrieveSymbolsFromSymtab(const llvm::object::ObjectFile& file, Args... symbols) {
    return NPerforator::NELF::NPrivate::RetrieveSymbolsFromSymtab(file, {symbols...});
}

template <typename... Args>
TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, Args... symbols) {
    return NPerforator::NELF::NPrivate::RetrieveSymbols(file, {symbols...});
}


TMaybe<llvm::object::SectionRef> GetSection(const llvm::object::ObjectFile& file, TStringBuf sectionName);

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location,
    TStringBuf sectionName
);

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromTextSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
);

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromRodataSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
);

bool IsElfFile(const llvm::object::ObjectFile& file);

} // namespace NPerforator::NELF
