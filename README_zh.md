# 辅助构建 mindspore serving 推理服务镜像

## 初始化工作目录

```bash
mkdir test_workspace

cd test_workspace

packctl init face-recognition --version 0.0.1
```

## 创建模型文件夹
以创建 resnet50 推理服务为例

```bash
mkdir /root/test_workspace/resnet50 
```

按照 mindspore 模型文件加载路径（参考 <https://gitee.com/mindspore/serving/tree/master/example/resnet>）放置模型文件, 放入文件之后的结构如下
```bash
root@sethostname:~/test_workspace/resnet50# tree -a
.
├── 1
│   └── resnet50_1b_cifar10.mindir
└── servable_config.py
```

提供模型服务的 API 说明文档:
1. 创建一个 markdown 文件，在起文件名的时候要注意，需要以 "method_" 前缀为开头，如 “method_client.md”
2. 对上面创建的文件使用 markdown 格式进行编辑
3. 将文件放入相关的模型文件夹内
4. 目前每个模型都至少需要一个文档文件，否则在前端页面展示会有一些问题

如最简单的文档 method_classify_top1.md:
```python
def run_classify_top1(method_name):
    """Client for servable resnet50 and method classify_top1[v1,v2,v3]"""
    print(f"\n--------------run_{method_name}----------")
    client = Client("localhost:5500", "resnet50", method_name)
    instances = []
    image_files, images_buffer = read_images()
    for image in images_buffer:
    instances.append({"image": image})
    result = client.infer(instances)
    print(result)
    for file, label in zip(image_files, result):
    print(f"{file}, label: {label['label']}")
```

加入之后的工作目录下 resnet50 模型目录下结构
```bash
root@sethostname:~/test_workspace/resnet50# tree
.
├── 1
│   └── resnet50_1b_cifar10.mindir
├── method_classify_top1.md
└── servable_config.py
```

## 构建镜像
构建镜像时会基于 baseImage 即已经准备了 mindspore-serving 运行环境的基础镜像，
向其注入上面我们当前工作目录中的所有文件，即模型文件和所有的配置文件

如果使用的 registry 是私有仓库，则需要提前登陆，避免构建镜像后上传时失败（使用 docker 且已登陆私有仓库可以跳过）
以登陆测试 registry xxx.thingsdao.com 为例
登陆成功后会提示成功
```bash
root@sethostname:~/test_workspace# packctl login xxx.thingsdao.com
Enter Username: admin
Enter Password: 
login registry [xxx.thingsdao.com] successfully!
```

使用命令构建镜像
```bash
packctl build xxx.thingsdao.com/edgewize/edgewize-model:v0.0.1 --baseImage xxx.thingsdao.com/ascendhub/base-server:v0.0.1 
```
