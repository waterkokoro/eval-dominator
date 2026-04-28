from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def main() -> None:
    core_dir = Path(__file__).resolve().parents[1]
    repo_dir = core_dir.parent
    proto_file = repo_dir / "proto" / "eval_service.proto"
    output_dir = core_dir / "src" / "opencompass_core" / "grpc" / "gen"
    output_dir.mkdir(parents=True, exist_ok=True)

    command = [
        sys.executable,
        "-m",
        "grpc_tools.protoc",
        f"-I{repo_dir / 'proto'}",
        f"--python_out={output_dir}",
        f"--grpc_python_out={output_dir}",
        str(proto_file),
    ]

    subprocess.run(command, check=True)
    print(f"已生成 Python gRPC 代码到: {output_dir}")


if __name__ == "__main__":
    main()
