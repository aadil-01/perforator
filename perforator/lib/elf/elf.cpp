#include "elf.h"


#include <llvm/Object/ELFObjectFile.h>

namespace NPerforator::NELF::NPrivate {

TMaybe<TSymbolMap> RetrieveSymbolsFromDynsym(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        return ParseDynsym(elf, symbols);
    });
}

TMaybe<TSymbolMap> RetrieveSymbolsFromSymtab(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        return ParseSymtab(elf, symbols);
    });
}

TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile &file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        TSymbolMap res = ParseDynsym(elf, symbols);

        llvm::SmallVector<TStringBuf> symtabSymbols;
        symtabSymbols.reserve(symbols.size());
        for (const TStringBuf& symbol : symbols) {
            if (!res.contains(symbol)) {
                symtabSymbols.push_back(symbol);
            }
        }

        TSymbolMap symtab = ParseSymtab(elf, symtabSymbols);
        for (auto& [key, value] : symtab) {
            res[key] = std::move(value);
        }

        return res;
    });
}

template <typename ELFT>
bool IsElfFileImpl(const llvm::object::ObjectFile& file) {
    return llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(&file);
}

} // namespace NPerforator::NELF::NPrivate

namespace NPerforator::NELF {

TMaybe<llvm::object::SectionRef> GetSection(const llvm::object::ObjectFile& file, TStringBuf sectionName) {
    for (const auto& section : file.sections()) {
        Y_LLVM_UNWRAP(name, section.getName(), { continue; });
        if (TStringBuf{name.data(), name.size()} == sectionName) {
            return section;
        }
    }

    return Nothing();
}

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location,
    TStringBuf sectionName
) {
    auto section = GetSection(file, sectionName);
    if (!section) {
        return Nothing();
    }

    Y_LLVM_UNWRAP(sectionData, section->getContents(), { return Nothing(); });

    if (location.Address < section->getAddress()) {
        return Nothing();
    }

    ui64 offset = location.Address - section->getAddress();
    if (offset >= sectionData.size()) {
        return Nothing();
    }

    size_t contentSize = Min<size_t>(location.Size, sectionData.size() - offset);

    return MakeMaybe(TConstArrayRef<ui8>(
        static_cast<const ui8*>(reinterpret_cast<const unsigned char*>(sectionData.data()) + offset),
        contentSize
    ));
}

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromTextSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
) {
    return RetrieveContentFromSection(file, location, NSections::kTextSectionName);
}

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromRodataSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
) {
    return RetrieveContentFromSection(file, location, NSections::kRoDataSectionName);
}

bool IsElfFile(const llvm::object::ObjectFile &file) {
#define TRY_ELF_TYPE(ELFT)                 \
if (NPrivate::IsElfFileImpl<ELFT>(file)) { \
    return true;                           \
}

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

#undef TRY_ELF_TYPE
    return false;
}

} // namespace NPerforator::NELF
