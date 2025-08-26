import os.path
import sys
import uuid
from datetime import datetime, UTC
from pprint import pformat

import library.python.fs

from build.plugins.lib.nots.package_manager import constants as pm_constants, utils as pm_utils
from devtools.frontend_build_platform.libraries.logging import init_logging, timeit_options
from devtools.frontend_build_platform.nots.builder.api import BuildError
from devtools.frontend_build_platform.nots.builder.cli.cli_args import AllOptions, get_args_parser, parse_args


def on_crash(exctype, value, traceback):
    if issubclass(exctype, BuildError):
        print(str(value), file=sys.stderr)
        sys.exit(value.code)
    else:
        sys.__excepthook__(exctype, value, traceback)


sys.excepthook = on_crash


def __add_uuid_for_output(bindir: str, output_file: str):
    uuid_file_name = f'{bindir}/{pm_constants.OUTPUT_TAR_UUID_FILENAME}'

    with open(uuid_file_name, 'w') as f:
        output_filename = os.path.basename(output_file)
        uuid_str = uuid.uuid1().hex
        timestamp = datetime.now(UTC).isoformat()

        f.write(f"{output_filename}: {uuid_str} - {timestamp}")


def __produce_old_output_tar(output_file: str):
    # TODO FBP-1978 (remove the function)
    old_output_tar_file = os.path.join(os.path.dirname(output_file), 'output.tar')

    library.python.fs.hardlink_or_copy(output_file, old_output_tar_file)


def _postprocess_output(args: AllOptions) -> None:
    with_after_build = getattr(args, 'with_after_build', False)
    output_file = getattr(args, 'output_file', args.node_modules_bundle)

    if args.command == 'build-package' and not with_after_build:
        output_file = args.node_modules_bundle

    if output_file and os.path.isfile(output_file):
        if output_file != args.node_modules_bundle:
            # TODO FBP-1978 (remove call)
            __produce_old_output_tar(output_file)
            __add_uuid_for_output(args.bindir, output_file)


# @timeit тут нельзя, т.к. измерение включается внутри
def main():
    args_parser = get_args_parser()
    args: AllOptions = parse_args(args_parser)

    if args.verbose:
        sys.stderr.write(
            f"Raw command string:\n\n{' '.join(sys.argv)}\n\nParsed arguments:\n\n{pformat(vars(args))}\n\n"
        )

    if args.local_cli:
        timeit_options.enable(silent=True, use_dumper=True, use_stderr=True)

    init_logging(args.verbose)

    args.func(args)

    _postprocess_output(args)

    if args.local_cli:
        dir_name = pm_utils.build_traces_store_path(args.moddir)
        timeit_options.dump_trace(
            os.path.join(dir_name, f'{args.command}.builder.trace.json'), otherData=dict(moddir=args.moddir)
        )


if __name__ == "__main__":
    main()
